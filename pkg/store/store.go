package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/containerd/v2/pkg/identifiers"

	"github.com/AkihiroSuda/alcless/pkg/userutil"
)

type Instance struct {
	Name string `json:"name"`
	User string `json:"user"`
}

func Instances(ctx context.Context) ([]Instance, error) {
	users, err := userutil.Users(ctx)
	if err != nil {
		return nil, err
	}
	var res []Instance
	for _, u := range users {
		if !strings.HasPrefix(u, userutil.Prefix) {
			continue
		}
		instName := userutil.InstanceFromUser(u)
		if err = ValidateName(instName); err != nil {
			return res, err
		}
		res = append(res, Instance{Name: instName, User: u})
	}
	return res, nil
}

func ValidateName(name string) error {
	const reserved = "alcless_"
	if strings.HasPrefix(name, reserved) {
		return fmt.Errorf("instance name must not start with %q", reserved)
	}
	if err := identifiers.Validate(name); err != nil {
		return err
	}
	return nil
}
