package main

import (
"flag"
"fmt"
"os"

"github.com/lecodev-26/sentinelflow/internal/version"
)

func main() {
showVersion := flag.Bool("version", false, "Show version")
flag.Parse()

if *showVersion {
fmt.Printf("sfctl %s\n", version.Full())
return
}

// Comandos básicos
args := flag.Args()
if len(args) == 0 {
fmt.Println("sfctl - SentinelFlow CLI")
fmt.Println()
fmt.Println("Usage:")
fmt.Println("  sfctl <command> [args]")
fmt.Println()
fmt.Println("Commands:")
fmt.Println("  version       Show version")
fmt.Println("  health        Check gateway health")
fmt.Println("  org list      List organizations")
fmt.Println("  project list  List projects")
fmt.Println("  provider list List providers")
fmt.Println("  model list    List models")
fmt.Println("  policy list   List policies")
fmt.Println()
fmt.Println("Version:", version.Full())
os.Exit(0)
}

switch args[0] {
case "version":
fmt.Println(version.Full())
case "health":
fmt.Println("Checking health... (not implemented yet)")
default:
fmt.Printf("Unknown command: %s\n", args[0])
os.Exit(1)
}
}
