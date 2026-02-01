package consumer

import (
	"crypto/sha256"
	"crypto/sha512"

	"github.com/xdg-go/scram"
)

// SHA256 and SHA512 hash generators for SCRAM authentication.
var (
	SHA256 scram.HashGeneratorFcn = sha256.New
	SHA512 scram.HashGeneratorFcn = sha512.New
)

// scramClient implements sarama.SCRAMClient for SCRAM authentication.
type scramClient struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

func (sc *scramClient) Begin(userName, password, authzID string) (err error) {
	sc.Client, err = sc.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	sc.ClientConversation = sc.Client.NewConversation()
	return nil
}

func (sc *scramClient) Step(challenge string) (response string, err error) {
	response, err = sc.ClientConversation.Step(challenge)
	return
}

func (sc *scramClient) Done() bool {
	return sc.ClientConversation.Done()
}
