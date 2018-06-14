package kubernetes

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test/ltops"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (c *Cluster) loadtestPod(pod string, resultsOutput io.Writer) error {
	commandOutputFile := filepath.Join(c.Configuration().WorkingDirectory, "results", "loadtest-out-"+pod+".txt")
	if err := os.MkdirAll(filepath.Dir(commandOutputFile), 0700); err != nil {
		return errors.Wrap(err, "unable to create results directory.")
	}
	outfile, err := os.Create(commandOutputFile)
	if err != nil {
		return errors.Wrap(err, "unable to create loadtest results file.")
	}

	cmd := exec.Command("kubectl", "exec", pod, "./bin/loadtest", "all")

	if resultsOutput != nil {
		cmd.Stdout = io.MultiWriter(outfile, resultsOutput)
	} else {
		cmd.Stdout = outfile
	}
	cmd.Stderr = outfile

	log.Info("Running loadtest on " + pod)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (c *Cluster) bulkLoad(loadtestPod string, appPod string, force bool) error {
	if c.Configuration().BulkLoadComplete && !force {
		log.Info("Bulk loading previously completed, skipping (use --force-bulk-load to force)")
		return nil
	}

	log.Info("Bulk importing data, this may take some time")
	cmd := exec.Command("kubectl", "exec", loadtestPod, "./bin/loadtest", "genbulkload")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Unfortunately kubectl cp doesn't work directly between pods
	cmd = exec.Command("kubectl", "cp", loadtestPod+":/mattermost-load-test/loadtestbulkload.json", c.Configuration().WorkingDirectory)
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("kubectl", "cp", filepath.Join(c.Configuration().WorkingDirectory, "loadtestbulkload.json"), appPod+":/mattermost/")
	if err := cmd.Run(); err != nil {
		return err
	}

	// If this command fails, assume user was already created
	cmd = exec.Command("kubectl", "exec", appPod, "--", "./bin/platform", "user", "create", "--email", "success+ltadmin@simulator.amazonses.com", "--username", "ltadmin", "--password", "ltpassword", "--system_admin")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Info(fmt.Sprintf("system admin already created or failed to create err=%v output=%v", err, string(out)))
	}

	cmd = exec.Command("kubectl", "exec", appPod, "--", "./bin/platform", "import", "bulk", "--workers", "64", "--apply", "./loadtestbulkload.json")
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrap(err, "bulk import failed: "+string(out))
	}

	// TODO: uncomment when post loading is sped up
	/*
		cmd = exec.Command("kubectl", "exec", loadtestPod, "./bin/loadtest", "loadposts")
		if out, err := cmd.CombinedOutput(); err != nil {
			return errors.Wrap(err, "loading posts failed: "+string(out))
		}
	*/

	c.Config.BulkLoadComplete = true
	err = saveCluster(c, c.Config.WorkingDirectory)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) Loadtest(options *ltops.LoadTestOptions) error {
	loadtestPods, err := c.GetLoadtestInstancesAddrs()
	if err != nil || len(loadtestPods) <= 0 {
		return errors.Wrap(err, "unable to get loadtest pods")
	}

	appPods, err := c.GetAppInstancesAddrs()
	if err != nil || len(appPods) <= 0 {
		return errors.Wrap(err, "unable to get app pods")
	}

	err = c.bulkLoad(loadtestPods[0], appPods[0], options.ForceBulkLoad)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(loadtestPods))

	for i, pod := range loadtestPods {
		pod := pod
		go func() {
			var err error
			if i == 0 {
				err = c.loadtestPod(pod, options.ResultsWriter)
			} else {
				err = c.loadtestPod(pod, nil)
			}
			if err != nil {
				log.Error(err)
			}
			wg.Done()
		}()
		// Give some time between instances just to avoid any races
		time.Sleep(time.Second * 10)
	}

	log.Info("Wating for loadtests to complete. See: " + filepath.Join(c.Configuration().WorkingDirectory, "results") + " for results.")
	wg.Wait()

	return nil
}
