# Noob Channel Server

Run a server which manages noob channels.  Clients can contact the bot to receive a channel which other noobs will also receive.

## Use

### Building

To build the binary, use the go build tool with appropriate flags for your desired system and architecture.

`GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o run/nc.binary main.go`

### Running

To run the binary, simply call it with a config file.  See below for more info on the configuration.

`./nc.binary -c nc.yaml`

### Configuration

The binary accepts a yaml config file.

```yaml
logLevel: 1
log: "nc.log"
ndf: "ndf.json"
storage: "ncBlob"
password: "password"
adminKeys: "ncAdminKeys"
contact: "ncContact.json"
```

### Contacting the bot

On initialization, the binary will start a client following the network.  The 
identity will be output to the path defined by the `contact` configuration field.  

The client can contact the bot using single use requests with the tag `noobChannel`.
