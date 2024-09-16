package steam

import (
	"errors"
	"os"
	"path"
	"sort"
	"strings"
)

// CompatTool holds info about a compatibility tool (like human-readable name
// and a list of the games that are using it)
type CompatTool struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Games     Games  `json:"games"`
	IsDefault bool   `json:"isDefault"`
	IsCustom  bool   `json:"isCustom"`
}

// CompatTools maps compatibility tool version IDs to CompatTool objects
// containing info like name and games.
type CompatTools map[string]*CompatTool

// IsInstalled returns if a tool is installed
func (t CompatTool) IsInstalled(s *Steam) bool {
	// TODO Implement for non custom tools
	if !t.IsCustom {
		return false
	}

	file, err := os.Open(path.Join(s.GetCompatibilityToolsDir(), t.ID))
	if err != nil {
		return false
	}

	fInfo, err := file.Stat()
	if err != nil || !fInfo.IsDir() {
		return false
	}

	return true
}

// IsValid checks whether a version ID (v) exists in the CompatTools config
func (c CompatTools) IsValid(v string) bool {
	for id := range c {
		if id == v {
			return true
		}
	}

	return false
}

// Add adds an entry to the CompatTools config (by tool id and (display) name)
func (c CompatTools) Add(id, name string) {
	if c[id] != nil {
		if c[id].Name != name && name != "" {
			c[id].Name = name
		}

		return
	}

	c[id] = &CompatTool{
		ID:        id,
		Name:      name,
		IsDefault: id == "",
		IsCustom:  id != "" && id == name,
		Games:     make(Games),
	}
}

// AddGame adds a Game entry to an existing CompatTool entry
func (c CompatTools) AddGame(toolID string, game *Game) bool {
	if c[toolID] == nil {
		return false
	}

	c[toolID].Games[game.Name] = game
	return true
}

// ReadCompatTools reads compatibility tool mappings from config, and sets
// s.CompatTools accordingly.
func (s *Steam) ReadCompatTools() error {
	tools, err := s.CompatToolMapping.ReadCompatTools()
	if err != nil {
		return err
	}

	games, err := s.LocalConfig.GetGames(s.EnableViewedSteamPlay)
	if err != nil {
		return err
	}

	def := tools.GetDefault()
	if def != nil {
		for _, game := range games {
			if tools.Includes(game.ID) {
				continue
			}

			tools.AddGame(def.ID, game)
		}
	}

	files, err := os.ReadDir(s.GetCompatibilityToolsDir())
	if err != nil {
		// If this directory does not exist, that's actually fine.
		// It can be treated as an empty folder in that scenario.
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	for _, file := range files {
		id := file.Name()
		if strings.HasPrefix(id, ".") {
			continue
		}
		tools.Add(id, id)
	}

	s.CompatTools = tools
	return nil
}

// Merge adds all entries from other to c (without duplicates)
func (c CompatTools) Merge(other *CompatTools) CompatTools {
	for _, tool := range *other {
		for _, game := range tool.Games {
			c.Add(tool.ID, tool.Name)
			c.AddGame(tool.ID, game)
		}
	}

	return c
}

// GetDefault returns the first entry from c that is a default compatibility tool
func (c CompatTools) GetDefault() *CompatTool {
	for _, tool := range c {
		if tool.IsDefault {
			return tool
		}
	}

	return nil
}

// Sort returns slice of alphabetically sorted CompatTools IDs
func (c CompatTools) Sort() []string {
	type kv struct {
		key   string
		value *CompatTool
	}

	var tmp []kv
	for k, v := range c {
		tmp = append(tmp, kv{k, v})
	}

	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i].value.Name < tmp[j].value.Name
	})

	var ids []string
	for _, tool := range tmp {
		ids = append(ids, tool.key)
	}

	return ids
}

// Includes returns whether a game is included in c
func (c CompatTools) Includes(appID string) bool {
	for _, tool := range c {
		if tool.Games.Includes(appID) {
			return true
		}
	}

	return false
}

// NewCompatTools returns new CompatTools struct (optionally initialized with
// data, which is a map from compatibility tool version IDs to slices of game
// IDs)
func (s *Steam) NewCompatTools(data ...map[string][]string) (*CompatTools, error) {
	compatTools := make(CompatTools)

	if len(data) > 0 {
		for versionID, games := range data[0] {
			for _, gameID := range games {
				game, _, err := s.GetGameData(gameID)
				if err != nil {
					return nil, err
				}

				versionName, _ := s.GetCompatToolName(versionID)

				compatTools.Add(versionID, versionName)
				compatTools.AddGame(versionID, game)
			}
		}
	}

	return &compatTools, nil
}
