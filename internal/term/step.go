package term

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh/spinner"
)

type Step struct {
	Name        string
	Run         func() (string, error)
	Interactive bool
}

func RunSteps(steps []Step) error {
	for _, s := range steps {
		if err := runStep(s); err != nil {
			return err
		}
	}
	return nil
}

func runStep(s Step) error {
	if s.Interactive {
		fmt.Println(Dim.Render("  · " + s.Name))
		result, err := s.Run()
		if err != nil {
			fmt.Printf("  %s %s\n", CrossMark, s.Name)
			return err
		}
		if strings.HasPrefix(result, "skipped") {
			fmt.Printf("  %s %s (%s)\n", WarnMark, s.Name, result)
		} else {
			fmt.Printf("  %s %s\n", CheckMark, s.Name)
		}
		return nil
	}

	var result string
	var runErr error
	err := spinner.New().
		Title("  " + s.Name).
		Action(func() {
			result, runErr = s.Run()
		}).
		Run()
	if err != nil {
		return err
	}
	if runErr != nil {
		fmt.Printf("  %s %s\n", CrossMark, s.Name)
		return runErr
	}

	if strings.HasPrefix(result, "skipped") {
		fmt.Printf("  %s %s (%s)\n", WarnMark, s.Name, result)
	} else {
		fmt.Printf("  %s %s\n", CheckMark, s.Name)
	}
	return nil
}
