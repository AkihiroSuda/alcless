package userutil

import (
	"errors"
	"os"
	"os/user"
	"strings"
)

const envGroup = "ALCLESS_GROUP"

var Mode string
var Prefix string
var groupName string

func init() {
	if groupName = os.Getenv(envGroup); groupName != "" {
		Mode = "group"
	} else {
		Mode = "prefix"
		Prefix = "alcless_" + me() + "_"
	}
}

func GroupName() string {
	return groupName
}

func me() string {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	if u.Username == "" {
		panic("no username")
	}
	return u.Username
}

func UserFromInstance(instName string) string {
	if Mode == "group" {
		return instName
	}
	return Prefix + instName
}

func InstanceFromUser(username string) string {
	if Mode == "group" {
		return username
	}
	return strings.TrimPrefix(username, Prefix)
}

func Exists(name string) (bool, error) {
	if _, err := user.Lookup(name); err != nil {
		var uee user.UnknownUserError
		if errors.As(err, &uee) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
