package noobChannel

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/single"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"time"
)

const receptionIdentityKey = "receptionIdentity"
const singleUseTag = "noobChannel"

// Init handles initializing or loading the variables used for the noob
// channel manager, connecting to the cmix network and registering callbacks.
func Init(ndfJson, storageDir, contactOutputPath string, password []byte,
	adminKeysDir string, rng *fastRNG.StreamGenerator) (*manager, error) {
	m := &manager{
		adminKeysDir: adminKeysDir,
	}

	// Init cmix object
	err := m.initCmix(ndfJson, storageDir, contactOutputPath, password)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to initialize cmix")
	}

	s := m.e2eClient.GetStorage()

	// Get channel manager identity, or create if needed
	var channelIdentity cryptoChannel.PrivateIdentity
	channelIdentityObj, err := s.Get(channelManagerIdentityKey)
	if err != nil {
		channelIdentity, err = cryptoChannel.GenerateIdentity(rng.GetStream())
		if err != nil {
			return nil, errors.WithMessage(err, "failed to generate new channel identity for manager")
		}
		marshalledChannelIdentity := channelIdentity.Marshal()
		err = s.Set(channelManagerIdentityKey, &versioned.Object{
			Version:   0,
			Timestamp: time.Now(),
			Data:      marshalledChannelIdentity,
		})
		if err != nil {
			return nil, errors.WithMessage(err, "failed to save new channel manager identity")
		}
	} else {
		channelIdentity, err = cryptoChannel.UnmarshalPrivateIdentity(channelIdentityObj.Data)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to unmarshal stored channel manager identity")
		}
	}

	obj, err := s.Get(channelCountKey)
	if err != nil {
		m.channelCount = 0
	} else {
		m.channelCount = binary.BigEndian.Uint64(obj.Data)
	}

	obj, err = s.Get(inCurrentChannelKey)
	if err != nil {
		m.inCurrentChannel = 0
	} else {
		m.inCurrentChannel = binary.BigEndian.Uint64(obj.Data)
	}

	obj, err = s.Get(currentChannelKey)
	if err != nil {
		err = m.generateNewChannel()
		if err != nil {
			return nil, errors.WithMessage(err, "failed to generate new channel")
		}
	} else {
		m.currentChannelInfo, err = broadcast.UnmarshalChannel(obj.Data)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to unmarshal current channel info")
		}
	}

	return m, nil
}

// initCmix initializes cmix managers and callbacks.
func (m *manager) initCmix(ndfJson, storageDir, contactOutput string, password []byte) error {
	if !utils.Exists(storageDir) {
		err := xxdk.NewCmix(ndfJson, storageDir, password, "")
		if err != nil {
			return errors.WithMessage(err, "failed to create new Cmix")
		}
	}

	cmix, err := xxdk.LoadCmix(storageDir, password, xxdk.GetDefaultCMixParams())
	if err != nil {
		return errors.WithMessage(err, "failed to load cmix")
	}

	receptionIdentity, err := xxdk.LoadReceptionIdentity(receptionIdentityKey, cmix)
	if err != nil {
		receptionIdentity, err = xxdk.MakeReceptionIdentity(cmix)
		if err != nil {
			return errors.WithMessage(err, "failed to make reception identity for cmix")
		}
		err = xxdk.StoreReceptionIdentity(receptionIdentityKey, receptionIdentity, cmix)
		if err != nil {
			return errors.WithMessage(err, "failed to store new cmix reception identity")
		}
	}

	m.e2eClient, err = xxdk.Login(cmix, m, receptionIdentity, xxdk.GetDefaultE2EParams())
	if err != nil {
		return errors.WithMessage(err, "failed to login to cmix")
	}

	dhPriv, err := receptionIdentity.GetDHKeyPrivate()
	m.singleListener = single.Listen(singleUseTag, receptionIdentity.ID, dhPriv, m.e2eClient.GetCmix(), m.e2eClient.GetStorage().GetE2EGroup(), m)

	lId := m.e2eClient.GetE2E().RegisterListener(&id.ZeroUser, catalog.XxMessage, m)
	fmt.Println(lId)

	receptionContact := m.e2eClient.GetReceptionIdentity().GetContact().Marshal()
	err = utils.WriteFile(contactOutput, receptionContact, os.ModePerm, os.ModePerm)
	if err != nil {
		return errors.WithMessage(err, "failed to write contact to file")
	}

	err = m.e2eClient.StartNetworkFollower(10 * time.Second)
	if err != nil {
		return errors.WithMessage(err, "failed to start follower")
	}

	return nil
}
