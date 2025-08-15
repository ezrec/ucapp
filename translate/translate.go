package translate

import (
	"log"

	"github.com/jeandeaual/go-locale"

	"golang.org/x/text/message"
)

var printer *message.Printer

func init() {
	locales, err := locale.GetLocales()
	if err != nil {
		log.Printf("apmos: locale: %v", err)
	}

	if len(locales) == 0 {
		locales = []string{"en-US"}
	}

	printer = message.NewPrinter(message.MatchLanguage(locales...))
}

// From an en-US Sprintf() format, translate to string.
func From(key message.Reference, args ...any) string {
	return printer.Sprintf(key, args...)
}
