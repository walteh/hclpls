package diff

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/fatih/color"
	"github.com/k0kubun/pp/v3"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/pmezard/go-difflib/difflib"
)

func TypedDiffExportedOnly[T any](want T, got T) string {
	printer := pp.New()
	printer.SetExportedOnly(true)
	printer.SetColoringEnabled(false)
	return diffTyped(printer, want, got)
}

func TypedDiff[T any](want T, got T) string {
	printer := pp.New()
	printer.SetExportedOnly(false)
	printer.SetColoringEnabled(false)
	return diffTyped(printer, want, got)
}

func diffd(want string, got string) string {
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(want),
		B:        difflib.SplitLines(got),
		FromFile: "Expected",
		FromDate: "",
		ToFile:   "Actual",
		ToDate:   "",
		Context:  1,
	})

	return diff

}

func diffTyped[T any](printer *pp.PrettyPrinter, want T, got T) string {
	// Enable colors

	var abc string

	switch any(want).(type) {
	case reflect.Type:
		want := ConvolutedFormatReflectType(any(want).(reflect.Type))
		got := ConvolutedFormatReflectType(any(got).(reflect.Type))
		return diffTyped[string](printer, want, got)
	case reflect.Value:
		w := any(want).(reflect.Value)
		g := any(got).(reflect.Value)
		want := ConvolutedFormatReflectValue(w)
		got := ConvolutedFormatReflectValue(g)
		return diffTyped[any](printer, want, got)
	case string:
		abc = diffd(any(got).(string), any(want).(string))
	default:
		abc = diffd(printer.Sprint(got), printer.Sprint(want))
	}
	if abc == "" {
		return ""
	}

	prevNoColor := color.NoColor
	defer func() {
		color.NoColor = prevNoColor
	}()
	color.NoColor = false

	expectedPrefix := fmt.Sprintf("[%s] %s", color.New(color.FgRed, color.Bold).Sprint("exp"), color.New(color.Faint).Sprint(" -"))
	actualPrefix := fmt.Sprintf("[%s] %s", color.New(color.Bold, color.FgBlue).Sprint("act"), color.New(color.Faint).Sprint(" +"))

	abc = strings.ReplaceAll(abc, "--- Expected", fmt.Sprintf("%s %s [%s]", color.New(color.Faint).Sprint("---"), color.New(color.FgRed).Sprint("expected"), color.New(color.FgRed, color.Bold).Sprint("exp")))
	abc = strings.ReplaceAll(abc, "+++ Actual", fmt.Sprintf("%s %s [%s]", color.New(color.Faint).Sprint("+++"), color.New(color.FgBlue).Sprint("actual"), color.New(color.FgBlue, color.Bold).Sprint("act")))
	hit := false
	realign := []string{}
	for _, found := range strings.Split(abc, "\n") {
		if strings.HasPrefix(found, "-") {
			realign = append(realign, expectedPrefix+color.New(color.FgRed).Sprint(found[1:]))
		} else if strings.HasPrefix(found, "+") {
			realign = append(realign, actualPrefix+color.New(color.FgBlue).Sprint(found[1:]))
		} else {
			if hit {
				realign = append(realign, color.New(color.Faint).Sprint(strings.Repeat(" ", 8)+found))
			} else {
				realign = append(realign, color.New(color.Faint).Sprint(found))
			}
			if strings.HasPrefix(found, "@@") {
				hit = true
			}
		}
	}

	str := "\n"
	str += strings.Join(realign, "\n")

	return str
}

func SingleLineStringDiff(want string, got string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(want, got, false)
	return dmp.DiffPrettyText(diffs)
}
