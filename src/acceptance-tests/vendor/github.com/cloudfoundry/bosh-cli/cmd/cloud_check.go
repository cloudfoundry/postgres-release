package cmd

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshui "github.com/cloudfoundry/bosh-cli/ui"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

type CloudCheckCmd struct {
	deployment boshdir.Deployment
	ui         boshui.UI
}

func NewCloudCheckCmd(deployment boshdir.Deployment, ui boshui.UI) CloudCheckCmd {
	return CloudCheckCmd{deployment: deployment, ui: ui}
}

func (c CloudCheckCmd) Run(opts CloudCheckOpts) error {
	probs, err := c.deployment.ScanForProblems()
	if err != nil {
		return err
	}

	table := boshtbl.Table{
		Content: "problems",
		Header:  []string{"#", "Type", "Description"},
		SortBy:  []boshtbl.ColumnSort{{Column: 0, Asc: true}},
	}

	for _, p := range probs {
		table.Rows = append(table.Rows, []boshtbl.Value{
			boshtbl.NewValueInt(p.ID),
			boshtbl.NewValueString(p.Type),
			boshtbl.NewValueString(p.Description),
		})
	}

	c.ui.PrintTable(table)

	if len(probs) == 0 {
		return nil
	} else if opts.Report {
		return bosherr.Errorf("%d problem(s) found", len(probs))
	}

	var answers []boshdir.ProblemAnswer

	if opts.Auto {
		answers, err = c.defaultResolutions(probs)
		if err != nil {
			return err
		}
	} else {
		answers, err = c.askForResolutions(probs)
		if err != nil {
			return err
		}
	}

	err = c.ui.AskForConfirmation()
	if err != nil {
		return err
	}

	return c.deployment.ResolveProblems(answers)
}

func (c CloudCheckCmd) askForResolutions(probs []boshdir.Problem) ([]boshdir.ProblemAnswer, error) {
	var answers []boshdir.ProblemAnswer

	for _, prob := range probs {
		var opts []string

		for _, res := range prob.Resolutions {
			opts = append(opts, res.Plan)
		}

		chosenIndex, err := c.ui.AskForChoice(prob.Description, opts)
		if err != nil {
			return nil, err
		}

		answers = append(answers, boshdir.ProblemAnswer{
			ProblemID:  prob.ID,
			Resolution: prob.Resolutions[chosenIndex],
		})
	}

	return answers, nil
}

func (c CloudCheckCmd) defaultResolutions(probs []boshdir.Problem) ([]boshdir.ProblemAnswer, error) {
	var answers []boshdir.ProblemAnswer

	for _, prob := range probs {
		answers = append(answers, boshdir.ProblemAnswer{
			ProblemID:  prob.ID,
			Resolution: boshdir.ProblemResolutionDefault,
		})
	}

	return answers, nil
}
