package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"gopkg.in/mgo.v2/bson"

	"github.com/docker/libtrust"
)

func joseBase64UrlEncode(b []byte) []byte {
	return []byte(strings.TrimRight(base64.URLEncoding.EncodeToString(b), "="))
}

type jsHeader struct {
	JWK       json.RawMessage `json:"jwk,omitempty" bson:"jwk,omitempty"`
	Algorithm string          `json:"alg" json:"alg"`
	Chain     []string        `json:"x5c,omitempty" bson:"x5c,omitempty"`
}

type jsSignature struct {
	Header    jsHeader `json:"header" bson:"header"`
	Signature string   `json:"signature" bson:"signature"`
	Protected string   `json:"protected" bson:"protected"`
}

// JSONSignature represents a signature of a json object.
type JSONSignature struct {
	Payload    string        `json:"payload" bson:"payload"`
	Signatures []jsSignature `json:"signatures" bson:"signatures"`
	// indent       string
	// formatLength int
	// formatTail   []byte
}

type PayloadLibtrust struct {
	Payload    string      `json:"payload" bson:"payload"`
	Signatures interface{} `json:"signatures,omitempty" bson:"signatures,omitempty"`
}

type PayloadIdentities struct {
	Archive bool `json:"archive" bson:"archive"`
	Modify  bool `json:"modify" bson:"modify"`
	Remove  bool `json:"remove" bson:"remove"`
}

type Payload struct {
	ID          string `json:"id" bson:"id"`
	Owner       string `json:"owner" bson:"owner"`
	Permissions struct {
		Identities map[string]PayloadIdentities `json:"identities" bson:"identities"`
		Public     bool                         `json:"public" bson:"public"`
	} `json:"permissions" bson:"permissions"`
	Schema string      `json:"schema" bson:"schema"`
	Data   interface{} `json:"data" bson:"data"`
	// Version struct {
	// 	App struct {
	// 		Name    string `json:"name"`
	// 		URL     string `json:"url"`
	// 		Version string `json:"version"`
	// 	} `json:"app"`
	// 	Created  uint64 `json:"created"`
	// 	ID       string `json:"id"`
	// 	Message  string `json:"message"`
	// 	Received uint64 `json:"received"`
	// 	Removed  uint64 `json:"removed"`
	// 	Updated  uint64 `json:"updated"`
	// } `json:"version"`
	Signatures []jsSignature `json:"signatures,omitempty" bson:"signatures,omitempty"`
}

func (s *Payload) ToJSON() ([]byte, error) {
	payloadStr, err := json.MarshalIndent(s, "", "    ")
	return payloadStr, err
}

type Instance struct {
	ID      string    `json:"id" bson:"_id"`
	Owner   *Identity `json:"owner" bson:"owner"`
	Payload Payload   `json:"payload" bson:"payload"`
}

func (s *Instance) ToJSON() ([]byte, error) {
	instanceStr, err := json.MarshalIndent(s, "", "    ")
	return instanceStr, err
}

func (s *Instance) GetProperJWS() (*libtrust.JSONSignature, error) {
	payloadJSON, _ := s.Payload.ToJSON()
	jws, err := libtrust.ParsePrettySignature(payloadJSON, "signatures")
	return jws, err
}

func (s *Instance) SetPayloadFromJson(jsonPayload []byte) error {
	return json.Unmarshal(jsonPayload, &s.Payload)
}

func (s *Instance) Sign() error {
	payload, err := s.Payload.ToJSON()
	if err != nil {
		log.Println("Could not encode payload")
		return err
	}
	js, err := libtrust.NewJSONSignature(payload)
	if err != nil {
		log.Println("Could not create jsign")
		return err
	}
	jwk, err := s.Owner.GetPrivateKey()
	if err != nil {
		return err
	}
	err = js.Sign(jwk)
	if err != nil {
		log.Println("Could not sign payload")
		return err
	}
	jsJSON, err := js.JWS()
	if err != nil {
		return err
	}
	tempJSONSignature := JSONSignature{}
	err = json.Unmarshal(jsJSON, &tempJSONSignature)
	if err != nil {
		return err
	}
	s.Payload.Signatures = tempJSONSignature.Signatures
	return nil
}

func (s *Instance) Verify() (bool, error) {
	jws, err := s.GetProperJWS()
	if err != nil {
		return false, err
	}
	_, err = jws.Verify()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Instance) Push() {
	// TODO Check if this is our own identity
	for identityHostname, _ := range s.Payload.Permissions.Identities {
		go func(instance *Instance) {
			// Get identity
			identity := Identity{}
			err := db.C("identities").Find(bson.M{"_id": identityHostname}).One(&identity)
			if err == nil {
				if identity.Hostname != "" && identity.GetURI() != "" {
					identity.Send(s)
				}
			} else {
				fmt.Println("Could not find local identity", identityHostname)
			}
		}(s)
	}
}
