package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/nning/protonutils/steam"
	"github.com/nning/protonutils/utils"
	"github.com/spf13/cobra"
)

var compatToolCmd = &cobra.Command{
	Use:   "compattool",
	Short: "Commands for management of compatibility tools",
}

var compatToolListCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List compatibility tools",
	Long:  "List compatibility tools.",
	Run:   compatToolList,
}

var compatToolSetCmd = &cobra.Command{
	Use:   "set [flags] <game> <version>",
	Short: "Set compatibility tool version for game",
	Long:  "Set compatibility tool version for game. Game search string can be app ID, game name, or prefix of game name. It is matched case-insensitively, first match is used. Version parameters have to be version IDs. See `compattool list` for list of possible options. If \"default\" is used as version, explicit mapping is removed and game uses default compatibility tool.",
	Args:  cobra.MinimumNArgs(2),
	Run:   compatToolSet,
}

var compatToolMigrateCmd = &cobra.Command{
	Use:   "migrate [flags] <fromVersion> <toVersion>",
	Short: "Migrate compatibility tool version mappings from on version to another",
	Long:  "Migrate compatibility tool version mappings from on version to another. Version parameters have to be version IDs. See `compattool list` for list of possible options.",
	Args:  cobra.MinimumNArgs(2),
	Run:   compatToolMigrate,
}

var compatToolCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Delete unused compatibility tools",
	Run:   compatToolClean,
}

func init() {
	rootCmd.AddCommand(compatToolCmd)

	compatToolCmd.AddCommand(compatToolListCmd)
	compatToolListCmd.Flags().BoolVarP(&ignoreCache, "ignore-cache", "c", false, "Ignore app ID/name cache")
	compatToolListCmd.Flags().StringVarP(&user, "user", "u", "", "Steam user name (or SteamID3)")

	compatToolCmd.AddCommand(compatToolSetCmd)
	compatToolSetCmd.Flags().BoolVarP(&ignoreCache, "ignore-cache", "c", false, "Ignore app ID/name cache")
	compatToolSetCmd.Flags().StringVarP(&user, "user", "u", "", "Steam user name (or SteamID3)")
	compatToolSetCmd.Flags().BoolVarP(&yes, "yes", "y", false, "Do not ask")

	compatToolCmd.AddCommand(compatToolMigrateCmd)
	compatToolMigrateCmd.Flags().BoolVarP(&yes, "yes", "y", false, "Do not ask")
	compatToolMigrateCmd.Flags().BoolVarP(&remove, "remove", "r", false, "Remove fromVersion after migration")

	compatToolCmd.AddCommand(compatToolCleanCmd)
	compatToolCleanCmd.Flags().BoolVarP(&ignoreCache, "ignore-cache", "c", false, "Ignore app ID/name cache")
	compatToolCleanCmd.Flags().StringVarP(&user, "user", "u", "", "Steam user name (or SteamID3)")
	compatToolCleanCmd.Flags().BoolVarP(&yes, "yes", "y", false, "Do not ask")
}

func validateVersion(vdf *steam.CompatToolMappingVdf, tools *steam.CompatTools, v string) {
	if strings.HasPrefix(v, "proton_") || tools.IsValid(v) || vdf.IsValid(v) {
		return
	}

	exitOnError(fmt.Errorf("Invalid version: %v", v))
}

func warnSteamRunning() {
	fmt.Printf("WARNING: Steam client seems to be running, changes will be overwritten on exit!\n\n")
}

func compatToolList(cmd *cobra.Command, args []string) {
	err := s.ReadCompatTools()
	exitOnError(err)

	for _, toolID := range s.CompatTools.Sort() {
		tool := s.CompatTools[toolID]
		games := tool.Games

		if tool.IsCustom && tool.IsInstalled(s) {
			fmt.Println(tool.ID)
			continue
		}

		for _, game := range games {
			if game.IsInstalled {
				id := ""
				if !tool.IsCustom && !tool.IsDefault {
					id = "[" + tool.ID + "]"
				}

				if tool.Name == "" {
					break
				}

				fmt.Println(tool.Name, id)
				break
			}
		}
	}
}

func compatToolSet(cmd *cobra.Command, args []string) {
	idOrName := args[0]
	newVersion := args[1]

	isRunning, _ := s.IsRunning()
	if isRunning {
		warnSteamRunning()
	}

	err := s.ReadCompatTools()
	exitOnError(err)

	id, name, err := s.GetAppIDAndName(idOrName)
	exitOnError(err)

	oldVersion := s.GetGameVersion(id)
	if oldVersion == nil {
		exitOnError(errors.New("Game not found"))
	}

	ctm := s.CompatToolMapping
	compatTools, err := ctm.ReadCompatTools()
	exitOnError(err)

	if newVersion == "default" {
		newVersion = ""
	} else {
		validateVersion(ctm, &compatTools, newVersion)
	}

	if oldVersion.ID == newVersion || oldVersion.Name == newVersion {
		fmt.Printf("%v is already using %v\n", name, newVersion)
		return
	}

	// TODO Warn about Steam overwriting the changes again

	fmt.Printf("WARNING: Make sure Steam is closed, otherwise changes will be overwritten on exit!\n\n")
	fmt.Println("App ID: ", id)
	fmt.Println("Name:   ", name)
	fmt.Println()
	if newVersion == "" {
		fmt.Println(oldVersion.Name, "->", "default")
	} else {
		fmt.Println(oldVersion.Name, "->", newVersion)
	}
	fmt.Println()

	if !yes {
		isOK, err := utils.AskYesOrNo("Really update?")
		exitOnError(err)

		if !isOK {
			fmt.Println("Aborted")
			return
		}
	}

	err = ctm.Update(id, newVersion)
	exitOnError(err)

	err = ctm.Save()
	exitOnError(err)

	// Update Steam cache
	err = s.ReadCompatTools()
	exitOnError(err)

	fmt.Println("Done")
}

func compatToolMigrate(cmd *cobra.Command, args []string) {
	fromVersion := args[0]
	toVersion := args[1]

	isRunning, _ := s.IsRunning()
	if isRunning {
		warnSteamRunning()
	}

	ctm := s.CompatToolMapping
	compatTools, err := ctm.ReadCompatTools()
	exitOnError(err)

	validateVersion(ctm, &compatTools, fromVersion)
	validateVersion(ctm, &compatTools, toVersion)

	version := compatTools[fromVersion]
	if version == nil || version.Games.CountInstalled() == 0 {
		exitOnError(fmt.Errorf("No installed games for %v", fromVersion))
	}

	fmt.Printf("WARNING: Make sure Steam is closed, otherwise changes will be overwritten on exit!\n\n")

	if toVersion == "" {
		fmt.Printf("%v -> %v\n\n", fromVersion, "default")
	} else {
		fmt.Printf("%v -> %v\n\n", fromVersion, toVersion)
	}

	// TODO Warn about Steam overwriting the changes again
	for _, game := range version.Games {
		if game.IsInstalled {
			fmt.Println("  * " + game.Name)
		}
	}
	fmt.Println()

	if !yes {
		isOK, err := utils.AskYesOrNo("Really update?")
		exitOnError(err)

		if !isOK {
			fmt.Println("Aborted")
			return
		}
	}

	for _, game := range compatTools[fromVersion].Games {
		if !game.IsInstalled {
			continue
		}

		err = ctm.Update(game.ID, toVersion)
		exitOnError(err)
	}

	err = ctm.Save()
	exitOnError(err)

	// Update Steam cache
	err = s.ReadCompatTools()
	exitOnError(err)

	if remove {
		fmt.Println()
		compatToolClean(cmd, []string{})
	}

	fmt.Println("Done")
}

func compatToolClean(cmd *cobra.Command, args []string) {
	err := s.ReadCompatTools()
	exitOnError(err)

	toDelete := utils.Slice[string]{}

	for id, tool := range s.CompatTools {
		hasInstalledGame := false

		for _, game := range tool.Games {
			if game.IsInstalled {
				hasInstalledGame = true
				break
			}
		}

		if !hasInstalledGame {
			toDelete = append(toDelete, id)
		}
	}

	dir := getCompatDir(s)
	files, err := os.ReadDir(dir)
	exitOnError(err)

	newToDelete := toDelete.Clone()
	for _, version := range toDelete {
		exists := false

		for _, file := range files {
			if file.Name() == version {
				exists = true
				break
			}
		}

		if !exists {
			newToDelete = newToDelete.DeleteValue(version)
		}
	}
	toDelete = newToDelete

	for _, file := range files {
		n := file.Name()
		if file.IsDir() && !strings.HasPrefix(n, ".") && s.CompatTools[n] == nil {
			toDelete = append(toDelete, n)
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("No unused compatibility tool found")
		return
	}

	fmt.Println("Unused versions found:")
	for _, version := range toDelete {
		fmt.Println("  * " + version)
	}
	fmt.Println()

	if !yes {
		isOK, err := utils.AskYesOrNo("Really delete?")
		exitOnError(err)

		if !isOK {
			fmt.Println("Aborted")
			return
		}
	}

	for _, version := range toDelete {
		err := os.RemoveAll(path.Join(dir, version))
		exitOnError(err)
	}

	fmt.Println("Done")
}
