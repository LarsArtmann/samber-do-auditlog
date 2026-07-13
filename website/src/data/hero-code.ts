import { siteConfig } from "./config";

const importPath = siteConfig.github.replace("https://github.com/", "github.com/");

export const heroCode = `package main

import (
    "os"

    auditlog "${importPath}"
    "github.com/samber/do/v2"
)

func main() {
    plugin, err := auditlog.New(auditlog.Config{
        Enabled:     true,
        ContainerID: "my-app",
    })
    if err != nil {
        panic(err)
    }

    injector := do.NewWithOpts(plugin.Opts())

    // Register and invoke services as usual
    do.Provide(injector, func(i do.Injector) (*Database, error) {
        return &Database{}, nil
    })
    do.MustInvoke[*Database](injector)

    // Export when ready
    plugin.ExportToHTML("audit.html")
    plugin.Report().WriteMermaid(os.Stdout)
}`;
