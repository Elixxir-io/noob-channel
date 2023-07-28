package noobChannel

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/single"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"path"
	"time"
)

const (
	noobChannelCap            = 100
	noobChannelName           = "NC_%s"
	noobChannelDescription    = "A channel for you super noobs that need some help"
	noobChannelSalt           = "i'm a little teapot short and stout"
	channelManagerIdentityKey = "managerIdentity"
	channelCountKey           = "channelCount"
	inCurrentChannelKey       = "inCurrentChannel"
	currentChannelKey         = "currentChannel"
)

// manager struct for handling noob channels
type manager struct {
	channelCount, inCurrentChannel uint64
	adminKeysDir                   string
	currentChannelInfo             *broadcast.Channel
	e2eClient                      *xxdk.E2e
	singleListener                 single.Listener
}

// respondToHello responds to a message from a client, returning a noob channel definition.
func (m *manager) respondToHello() ([]byte, error) {
	m.inCurrentChannel = m.inCurrentChannel + 1
	if err := m.saveCurrentPosition(); err != nil {
		m.inCurrentChannel = m.inCurrentChannel - 1
		return nil, errors.WithMessage(err, "failed to save new position")
	}
	if m.inCurrentChannel > noobChannelCap {
		m.channelCount = m.channelCount + 1
		if err := m.saveChannelCount(); err != nil {
			m.inCurrentChannel = m.inCurrentChannel - 1
			m.channelCount = m.channelCount - 1
			return nil, errors.WithMessage(err, "failed to save channel count")
		}
		err := m.generateNewChannel()
		if err != nil {
			m.inCurrentChannel = m.inCurrentChannel - 1
			m.channelCount = m.channelCount - 1
			return nil, errors.WithMessage(err, "Failed to generate new channel")
		}
	}

	return m.currentChannelInfo.Marshal()
}

// generateNewChannel generates a new noob channel, swapping it for the old one.
func (m *manager) generateNewChannel() error {
	h := hash.CMixHash.New()
	h.Write([]byte(fmt.Sprintf("%d", m.channelCount)))
	h.Write([]byte(noobChannelSalt))
	newChannelHashBytes := h.Sum(nil)
	newChannelCodename := codename.GenerateChannelCodename(newChannelHashBytes)

	channelName := fmt.Sprintf(noobChannelName, newChannelCodename)
	jww.INFO.Println(channelName)

	newChannel, privateKey, err := broadcast.NewChannel(channelName, noobChannelDescription, broadcast.Public, m.e2eClient.GetCmix().GetMaxMessageLength(), m.e2eClient.GetRng().GetStream())
	if err != nil {
		return errors.WithMessage(err, "failed to generate new noob channel")
	}

	m.currentChannelInfo = newChannel
	err = m.saveCurrentChannel()
	if err != nil {
		return errors.WithMessage(err, "failed to save new channel info")
	}
	err = m.writeAdminToDisk(newChannelCodename, newChannel, privateKey)
	if err != nil {
		return errors.WithMessage(err, "failed to write admin info to disk")
	}
	return nil
}

/* STORAGE FUNCTIONS */

// writeAdminToDisk writes admin info and channel definition to disk.
func (m *manager) writeAdminToDisk(channelHash string, ch *broadcast.Channel, pk rsa.PrivateKey) error {
	channelDir := path.Join(m.adminKeysDir, channelHash)
	err := os.Mkdir(channelDir, os.ModePerm)
	if err != nil {
		return errors.WithMessage(err, "failed to make directory for channel")
	}

	adminKeyBytes := pk.MarshalPem()
	channelInfoBytes, err := ch.Marshal()
	if err != nil {
		return errors.WithMessage(err, "failed to marshal channel info")
	}

	channelInfoFile := path.Join(channelDir, "channelInfo.json")
	err = utils.WriteFile(channelInfoFile, channelInfoBytes, os.ModePerm, os.ModePerm)
	if err != nil {
		return errors.WithMessagef(err, "failed to write channel info to file %s", channelInfoFile)
	}
	channelAdminFile := path.Join(channelDir, "channelAdmin.key")
	err = utils.WriteFile(channelAdminFile, adminKeyBytes, os.ModePerm, os.ModePerm)
	if err != nil {
		return errors.WithMessagef(err, "failed to write channel admin keys to file %s", channelAdminFile)
	}

	return nil
}

// saveCurrentPosition stores the inCurrentChannel value for the manager.
func (m *manager) saveCurrentPosition() error {
	s := m.e2eClient.GetStorage()

	obj := &versioned.Object{
		Version:   0,
		Timestamp: time.Now(),
		Data:      make([]byte, 8),
	}
	binary.BigEndian.PutUint64(obj.Data, m.inCurrentChannel)
	err := s.Set(inCurrentChannelKey, obj)
	if err != nil {
		return err
	}
	return nil
}

// saveChannelCount saves the channelCount stored on the manager.
func (m *manager) saveChannelCount() error {
	s := m.e2eClient.GetStorage()
	obj := &versioned.Object{
		Version:   0,
		Timestamp: time.Now(),
		Data:      make([]byte, 8),
	}
	binary.BigEndian.PutUint64(obj.Data, m.channelCount)
	err := s.Set(channelCountKey, obj)
	if err != nil {
		return err
	}
	return nil
}

// saveCurrentChannel saves the current noob channel definition.
func (m *manager) saveCurrentChannel() error {
	s := m.e2eClient.GetStorage()
	currentChannelBytes, err := m.currentChannelInfo.Marshal()
	if err != nil {

	}
	obj := &versioned.Object{
		Version:   0,
		Timestamp: time.Now(),
		Data:      currentChannelBytes,
	}
	err = s.Set(currentChannelKey, obj)
	if err != nil {
		return err
	}
	return nil
}
