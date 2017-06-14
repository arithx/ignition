package main

import (
	"flag"
	"fmt"
)

var (
	flagRoot         string
	flagHomeDir      string
	flagCreateHome   bool
	flagNoCreateHome bool
	flagNoUserGroup  bool
	flagSystem       bool
	flagNoLogInit    bool
	flagPassword     string
	flagUid          int
	flagComment      string
	flagGid          int
	flagGroups       string
	flagShell        string
)

func main() {
	flag.StringVar(&flagRoot, "root", "", "Apply changes in the CHROOT_DIR directory and use the configuration files from the CHROOT_DIR directory")
	flag.StringVar(&flagHomeDir, "home-dir", "", "The new user will be created using HOME_DIR as the value for the user's login directory")
	flag.BoolVar(&flagCreateHome, "--create-home", false, "Create the user's home directory if it does not exist.")
	flag.BoolVar(&flagNoCreateHome, "--no-create-home", false, "Do no create the user's home directory")
	flag.BoolVar(&flagNoUserGroup, "--no-user-group", false, "Do not create a group with the same name as the user")
	flag.BoolVar(&flagSystem, "--system", false, "Create a system account")
	flag.BoolVar(&flagNoLogInit, "--no-log-init", false, "Do not add the user to the lastlog and faillog databases")
	flag.StringVar(&flagPassword, "password", "", "The encrypted password, as returned by crypt")
	flag.IntVar(&flagUid, "uid", 0, "The numerical value of the user's ID")
	flag.StringVar(&flagComment, "comment", "", "Any text string. It is generally a short description of the login, and is currently used as the field for the user's full name.")
	flag.IntVar(&flagGid, "gid", 0, "The group name or number of the user's initial login group")
	flag.StringVar(&flagGroups, "groups", "", "A list of supplementary groups which the user is also a member of")
	flag.StringVar(&flagShell, "shell", "", "The name of the user's login shell")

	flag.Parse()

	fmt.Printf("stub for useradd call with the following arguments:\n")
	fmt.Printf("--root=%s\n", flagRoot)
	fmt.Printf("--uid=%d\n", flagUid)
	fmt.Printf("--gid=%d\n", flagGid)
	fmt.Printf("--password=%s\n", flagPassword)
	fmt.Printf("--home-dir=%s\n", flagHomeDir)
	fmt.Printf("--create-home=%t\n", flagCreateHome)
	fmt.Printf("--no-create-home=%t\n", flagNoCreateHome)
	fmt.Printf("--no-user-group=%t\n", flagNoUserGroup)
	fmt.Printf("--system=%t\n", flagSystem)
	fmt.Printf("--no-log-init=%t\n", flagNoLogInit)
	fmt.Printf("--comment=%s\n", flagComment)
	fmt.Printf("--groups=%s\n", flagGroups)
	fmt.Printf("--shell=%s\n", flagShell)
}
