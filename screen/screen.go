package screen

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/tiagomelo/ewma-policy-poc/screen/stats"
)

// Here we are embedding the banner
var (
	//go:embed banner.txt
	banner string
)

// screen implements Screen interface
type screen struct {
	areaPrinter *pterm.AreaPrinter
	layout      *pterm.CenterPrinter
}

// template is a helper function to define the
// layout
func template(name string, value string) string {
	const MAX_LENGTH int = 42
	pad := MAX_LENGTH - len(name)

	// Example output
	// [ Total                                  1883 ]
	return fmt.Sprintf("[ %s %*s ]", name, pad, value)
}

// NewScreen returns a new Screen
func New() (*screen, error) {
	area, err := pterm.DefaultArea.Start()
	if err != nil {
		return nil, errors.Wrap(err, "starting printer")
	}
	return &screen{area, new(pterm.CenterPrinter)}, nil
}

func (s *screen) UpdateContent(stats *stats.Statistics, finalUpdate bool) error {
	out := []string{
		template("Total DNS requests", fmt.Sprintf("%d", stats.TotalDnsRequests())),
		template("Total failed DNS requests", fmt.Sprintf("%d", stats.TotalFailedDnsRequests())),
		template("DNS requests/second", fmt.Sprintf("%d", stats.RequestsPerSecond())),
		template("Available DNS servers", fmt.Sprintf("%d", stats.TotalAvailableServers())),
		template("Unavailable DNS servers", fmt.Sprintf("%d", stats.TotalUnavailableServers())),
		template("Elapsed Time", formatDuration(stats.ElapsedTime())),
	}
	banner := pterm.DefaultCenter.Sprint(string(banner))
	content := s.layout.Sprint(strings.Join(out, "\n"))
	s.areaPrinter.Update(banner + content)
	if finalUpdate {
		if err := s.areaPrinter.Stop(); err != nil {
			return errors.Wrap(err, "stopping printer")
		}
	}
	return nil
}

// formatDuration formats a duration to the format "hh:mm:ss".
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02dh%02dm%02ds", h, m, s)
}
