package internal

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/earthboundkid/versioninfo/v2"
)

func ShowVersion() {
	log.Printf("Version: %s\n", versioninfo.Short())
}

func EnvironmentVars() {
	log.Println("Environment variables")

	sensitiveRegex := regexp.MustCompile(`(?i)(PASSWORD|API_KEY|ACCESS_KEY|SECRET)`)
	environ := os.Environ()
	sort.Slice(environ, func(i, j int) bool {
		keyI := strings.SplitN(environ[i], "=", 2)[0]
		keyJ := strings.SplitN(environ[j], "=", 2)[0]
		return keyI < keyJ
	})

	for _, entry := range environ {
		kv := strings.SplitN(entry, "=", 2)
		if sensitiveRegex.MatchString(kv[0]) {
			log.Printf("  %s: ********\n", kv[0])
		} else {
			log.Printf("  %s: %s\n", kv[0], kv[1])
		}
	}
}

func UserInfo() {
	log.Printf("PID: %d", os.Getpid())
	currentUser, err := user.Current()
	if err != nil {
		log.Printf("Error getting current user: %v", err)
	} else {
		log.Printf("User: uid=%s(%s) gid=%s", currentUser.Uid, currentUser.Username, currentUser.Gid)
	}
	groups, err := os.Getgroups()
	if err != nil {
		log.Printf("Error getting groups: %v", err)
	} else {
		groupNames := make([]string, 0, len(groups))
		for _, gid := range groups {
			group, err := user.LookupGroupId(strconv.Itoa(gid))
			if err != nil {
				groupNames = append(groupNames, strconv.Itoa(gid)) // Append ID if name lookup fails
			} else {
				groupNames = append(groupNames, fmt.Sprintf("%s(%s)", group.Name, group.Gid))
			}
		}
		log.Printf("Groups: %v", groupNames)
	}
}
