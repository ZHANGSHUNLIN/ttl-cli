package command

import (
	"fmt"
	"sort"
	"ttl-cli/db"
	"ttl-cli/i18n"
	"ttl-cli/models"

	"github.com/spf13/cobra"
)

var TagsCmd = &cobra.Command{
	Use:   "tags [tag]",
	Short: i18n.T("command.tags.short"),
	Long:  i18n.T("command.tags.long"),
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return runTagsList()
		}
		return runTagResources(args[0])
	},
}

func runTagsList() error {
	stats, err := db.GetTagStats()
	if err != nil {
		return fmt.Errorf(i18n.T("command.tags.error_fetch"), err)
	}

	if len(stats) == 0 {
		Println(i18n.T("command.tags.no_tags"))
		return nil
	}

	for _, stat := range stats {
		Printf("%s (%d)\n", stat.Tag, stat.Count)
	}

	return nil
}

func runTagResources(tag string) error {
	resources, err := db.GetAllResources()
	if err != nil {
		return fmt.Errorf(i18n.T("command.tags.error_fetch"), err)
	}

	var matchingKeys []models.ValJsonKey
	for key, val := range resources {
		if key.Type != models.ORIGIN {
			continue
		}
		for _, t := range val.Tag {
			if t == tag {
				matchingKeys = append(matchingKeys, key)
				break
			}
		}
	}

	if len(matchingKeys) == 0 {
		return fmt.Errorf(i18n.T("command.tags.not_found"), tag)
	}

	sort.Slice(matchingKeys, func(i, j int) bool {
		return resources[matchingKeys[i]].CreatedAt > resources[matchingKeys[j]].CreatedAt
	})

	Printf(i18n.T("command.tag_resources.header"), tag)
	for _, key := range matchingKeys {
		Println(fmt.Sprintf("  %s", key.Key))
	}
	Println(fmt.Sprintf(i18n.T("command.tag_resources.total"), len(matchingKeys)))

	return nil
}
