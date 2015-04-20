package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"gopkg.in/mgo.v2/bson"
)

func HandlePublicInit(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GET /init")

	hostnameKv := KV{}
	err := db.C("settings").Find(bson.M{"_id": "hostname"}).One(&hostnameKv)
	if err == nil {
		fmt.Println("Already active")
		return
	}

	var host string = r.Host
	if strings.Contains(host, ":") {
		host, _, _ = net.SplitHostPort(host)
	}

	fmt.Println("Current hostname is ", host)

	var identity Identity = Identity{}
	identity.Hostname = host
	identity.Init()
	fmt.Println(identity)
	// selfJSON, err := json.Marshal(&identity)
	// if err == nil {

	err = db.C("identities").Insert(&identity)
	if err != nil {
		fmt.Println(err)
	}

	hostnameKv.Key = "hostname"
	hostnameKv.Value = host
	err = db.C("settings").Insert(&hostnameKv)
	if err != nil {
		fmt.Println(err)
	}

	rt.self = &identity
	json.NewEncoder(w).Encode(rt.self)
	// }
}

func HandlePublicIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GET /")
	json.NewEncoder(w).Encode(rt.self)
}

func HandlePublicIndexPost(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POST /")

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	var identity Identity = Identity{}
	if err := json.Unmarshal(body, &identity); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	} else {
		rt.insertIdentity(&identity)
		json.NewEncoder(w).Encode(rt.self)
	}
}

func HandleIdentityInstancesPost(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POST /instances")

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	var instance Instance = Instance{}
	instance.SetPayloadFromJson(body)

	body, err = instance.ToJSON()
	if err == nil {
		fmt.Println(">>> GOT", string(body))
		valid, err := instance.Verify()
		if err == nil {
			if valid == true {
				fmt.Println(">>> IS VALID")
			} else {
				fmt.Println(">>> IS *NOT* VALID")
			}
		} else {
			fmt.Println(">>> error validating", err)
		}
	}
	json.NewEncoder(w).Encode(self)
}

func HandleOwnIdentities(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GET /identities")
	var identities []Identity
	err := db.C("settings").Find(bson.M{}).All(&identities)
	if err != nil {
		fmt.Println(err)
	}
	json.NewEncoder(w).Encode(identities)
}

func HandleOwnIdentitiesPost(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POST /identities")

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	var identity Identity = Identity{}
	if err := json.Unmarshal(body, &identity); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	} else {
		identity, err = FetchIdentity(identity.GetURI())
		// if err == nil {
		// 	rt.insertIdentity(&identity)
		// }
		err = db.C("identities").Insert(&identity)
		if err != nil {
			fmt.Println(err)
		}
		json.NewEncoder(w).Encode(identity)
	}
}
