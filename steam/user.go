package steam

import (
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/MrWaggel/gosteamconv"
)

func (s *Steam) getID64(u string) (string, error) {
	uid := s.LoginUsers.GetID64(u)
	if uid != "" {
		return uid, nil
	}

	return "", errors.New("User not found: " + u)
}

func (s *Steam) userToID32(u string) (string, error) {
	if u == "" {
		return "", nil
	}

	idStr64, err := s.getID64(u)
	if err != nil {
		x, e := strconv.ParseInt(u, 10, 32)
		if e != nil {
			return "", err
		}

		_, e = gosteamconv.SteamInt32ToString(int32(x))
		if e != nil {
			return "", err
		}

		return u, nil
	}

	idInt64, err := strconv.ParseInt(idStr64, 10, 64)
	if err != nil {
		return "", err
	}

	str, err := gosteamconv.SteamInt64ToString(idInt64)
	if err != nil {
		return "", err
	}

	idInt32, err := gosteamconv.SteamStringToInt32(str)
	if err != nil {
		return "", err
	}

	return fmt.Sprint(idInt32), nil
}

func (s *Steam) getUID(u string) (string, error) {
	dir := path.Join(s.Root, "userdata")
	// TODO Sort entries by last change?
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	uid := entries[0].Name()

	if len(entries) <= 1 {
		return uid, nil
	}

	users := make([]string, 0)

	for _, entry := range entries {
		name := entry.Name()
		if name == u {
			return name, nil
		}

		isEntryNumeric, err := regexp.MatchString("^[0-9]*$", name)
		if err != nil {
			return "", err
		}

		if name != "0" && isEntryNumeric {
			users = append(users, name)
		}
	}

	if len(users) == 0 {
		return "", errors.New("No user found")
	}

	uid = users[0]

	if len(users) > 1 {
		fmt.Fprintln(os.Stderr,
			"Warning: Several Steam users available, using "+uid+"\n"+
				"All available users: "+strings.Join(users, ", ")+"\n"+
				"Option \"-u\" can be used to specify user\n")
	}

	return uid, nil
}
