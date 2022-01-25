package steam2

import (
	"io/ioutil"
	"path"
)

// InnerOffsetShortcuts sets byte count before appid match in shortcuts
const InnerOffsetShortcuts = 1

func (s *Steam) initShortcuts() error {
	p := path.Join(s.Root, "userdata", s.UID, "config", "shortcuts.vdf")
	in, err := ioutil.ReadFile(p)
	if err != nil {
		return err
	}

	s.Shortcuts = &BinaryVdf{
		Bytes: in,
		Path:  p,
	}

	return nil
}
