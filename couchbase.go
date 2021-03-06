package securecouchbase

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

const NOT_FOUND string = "Not found"

func IsNotFoundError(err error) bool {
	// No error?
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), NOT_FOUND)
}

// EncryptedData a container for encrypted and signed data
type EncryptedData struct {
	EncryptedAndSigned []byte
}

// ProtectedDataSet a container for signed data
type ProtectedDataSet struct {
	Data      interface{}
	Signature []byte
}

// ProtectedDataRead a json.RawMessage wrapper for protected data
type ProtectedDataRead struct {
	Data      json.RawMessage
	Signature []byte
}

// SetWithEncryption encrypts data before sending it to couchbase
func SetWithEncryption(id string, exp int, object interface{}, connection Bucket, provider SecurityProvider) error {
	enc, err := json.Marshal(object)
	if err != nil {
		return err
	}

	encryptedBuffer := new(bytes.Buffer)
	err = provider.Encrypt(bytes.NewReader(enc), encryptedBuffer)
	if err != nil {
		return err
	}

	encryptedData := EncryptedData{encryptedBuffer.Bytes()}
	if err := connection.Set(id, exp, encryptedData); err != nil {
		return err
	}

	return nil
}

// GetWithEncryption decrypts encrypted data given a key in couchbase
func GetWithEncryption(id string, object interface{}, connection Bucket, provider SecurityProvider) error {
	var encryptedData EncryptedData
	err := connection.Get(id, &encryptedData)
	if err != nil {
		return err
	}

	decryptReader, err := provider.Decrypt(bytes.NewReader(encryptedData.EncryptedAndSigned))
	if err != nil {
		return err
	}

	buffer := new(bytes.Buffer)
	io.Copy(buffer, decryptReader)
	decryptReader.Close()
	json.Unmarshal(buffer.Bytes(), object)

	return nil
}

// SetWithSignature signs json structure before putting it in couchbase
func SetWithSignature(id string, exp int, object interface{}, connection Bucket, provider SecurityProvider) error {
	enc, err := json.Marshal(object)
	if err != nil {
		return err
	}

	sigBuffer := new(bytes.Buffer)
	err = provider.SignDetached(bytes.NewReader(enc), sigBuffer)
	if err != nil {
		return err
	}

	protectedData := ProtectedDataSet{object, sigBuffer.Bytes()}
	if err := connection.Set(id, exp, protectedData); err != nil {
		return err
	}

	return nil
}

// GetWithSignature verifys a json object with a detached signature
func GetWithSignature(id string, object interface{}, connection Bucket, provider SecurityProvider) error {
	var protectedData ProtectedDataRead
	err := connection.Get(id, &protectedData)
	if err != nil {
		return err
	}

	err = provider.VerifyDetached(bytes.NewReader(protectedData.Data), bytes.NewReader(protectedData.Signature))
	if err != nil {
		return err
	}

	json.Unmarshal(protectedData.Data, object)

	return nil
}
