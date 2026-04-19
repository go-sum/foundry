package build

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-sum/assets/config"
)

func DownloadFonts(cfg *config.Config, client *http.Client, out io.Writer) error {
	for _, dl := range cfg.Fonts.Downloads {
		version := ResolveVersion(dl.Name, dl.Version)
		url := strings.ReplaceAll(dl.URL, "{version}", version)
		downloaded, err := FetchURL(client, url, dl.Target, out)
		if err != nil {
			return fmt.Errorf("font download %s: %w", dl.Name, err)
		}
		if downloaded {
			fmt.Fprintf(out, "  ✓ downloaded font %s -> %s\n", dl.Name, dl.Target)
		}
	}
	return nil
}
