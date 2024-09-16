package steam

import (
	"path"

	"github.com/BenLubar/vdf"
	log "github.com/sirupsen/logrus"
)

// LocalConfigVdf represents parsed VDF config for app config from
// localconfig.vdf
type LocalConfigVdf struct {
	Vdf
}

// GetGames returns a slice of gamesr read from localconfig.vdf.
// The `enableViewedSteamPlay` argument controls whether the slice only contains
// games for which the user confirmed the Steam Play disclaimer
func (v *LocalConfigVdf) GetGames(enableViewedSteamPlay bool) ([]*Game, error) {
	log.Debug("LocalConfigVdf.GetGames()")

	games := make([]*Game, 0)
	var x *vdf.Node

	x = v.Node.FirstSubTree()

	for ; x != nil; x = x.NextChild() {
		id := x.Name()

		if enableViewedSteamPlay {
			viewedSteamPlay := x.FirstByName("ViewedSteamPlay").String()
			if viewedSteamPlay != "1" {
				continue
			}
		}

		game, isValid, err := v.Steam.GetGameData(id)
		if err != nil {
			return nil, err
		}

		if !isValid {
			continue
		}

		games = append(games, game)
	}

	return games, nil
}

func (s *Steam) initLocalConfig() error {
	p := path.Join(s.Root, "userdata", s.UID, "config", "localconfig.vdf")
	log.Debug("steam.initLocalConfig(", p, ")")

	n, err := ParseTextConfig(p)
	if err != nil {
		return err
	}

	key := []string{"Software", "Valve", "Steam", "apps"}
	x, err := Lookup(n, key)
	if err != nil {
		return err
	}

	s.LocalConfig = &LocalConfigVdf{Vdf{n, x, p, s}}

	return nil
}
