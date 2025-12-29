package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/greatbody/vb6-utf8-virtualization/internal/config"
	"github.com/greatbody/vb6-utf8-virtualization/internal/vfs"
	"github.com/stirante/dokan-go"
)

func main() {
	configPath := flag.String("config", "config.json", "Path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Printf("Could not load config: %v. Using default values.", err)
		cfg = config.DefaultConfig()
	}

	// Setup filter
	filter := vfs.NewFilter(cfg.AllowedProcesses, cfg.AllowedExtensions)

	// Setup VFS
	fs := vfs.NewProxyFS(cfg.PhysicalPath, filter)

	// Dokan options
	dokanCfg := &dokan.Config{
		Path:       cfg.MountPoint,
		FileSystem: fs,
	}

	// Mount
	fmt.Printf("Starting mount to %s using Dokany 1.x...\n", cfg.MountPoint)
	m, err := dokan.Mount(dokanCfg)
	if err != nil {
		log.Fatalf("Mount failed: %v\nMake sure Dokany 1.x driver is installed and DOKAN1.DLL is in your PATH.", err)
	}
	fmt.Printf("Mounted %s to %s successfully\n", cfg.PhysicalPath, cfg.MountPoint)

	// Wait for signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		fmt.Println("Signal received, unmounting...")
		m.Close()
	}()

	// Block until done
	if err := m.BlockTillDone(); err != nil {
		log.Printf("Dokan error: %v", err)
	}
	fmt.Println("Exiting.")
}
