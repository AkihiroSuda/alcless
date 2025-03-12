package userutil

import (
	"errors"
	"os/user"
	"strings"
)

// Prefix is the prefix of the user accounts.
var Prefix = "alcless_" + me() + "_"

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
	return Prefix + instName
}

func InstanceFromUser(username string) string {
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
