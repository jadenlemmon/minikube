//go:build !windows

/*
Copyright 2020 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package daemonize

import (
	"fmt"
	"os"
	"strconv"

	"github.com/VividCortex/godaemon"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"k8s.io/minikube/pkg/minikube/localpath"
)

type Daemon struct {
	Profile     string
	PidFileName string
	PidFilePath string
}

func (d *Daemon) kill() error {
	f, err := os.ReadFile(d.PidFilePath)
	if os.IsNotExist(err) {
		return nil
	}
	defer func() {
		if err := os.Remove(d.PidFilePath); err != nil {
			klog.Errorf("error deleting %s: %v, you may have to delete in manually", d.PidFilePath, err)
		}
	}()
	if err != nil {
		return errors.Wrapf(err, "reading %s", d.PidFilePath)
	}
	pid, err := strconv.Atoi(string(f))
	if err != nil {
		return errors.Wrapf(err, "converting %s to int", f)
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return errors.Wrap(err, "finding process")
	}
	klog.Infof("killing process %v as it is an old process", pid)
	if err := p.Kill(); err != nil {
		return errors.Wrapf(err, "killing %v", pid)
	}
	return nil
}

func (d *Daemon) start() error {
	if err := d.kill(); err != nil {
		return errors.Wrap(err, "daemonizing")
	}

	_, _, err := godaemon.MakeDaemon(&godaemon.DaemonAttr{})
	if err != nil {
		return err
	}

	// now that this process has daemonized, it has a new PID
	pid := os.Getpid()

	if err := os.WriteFile(d.PidFilePath, []byte(fmt.Sprintf("%v", pid)), 0600); err != nil {
		return err
	}

	return nil
}

func NewDaemon(profile string, pidName string) *Daemon {
	pidFileName := fmt.Sprintf("%v.pid", pidName)

	pidFilePath := localpath.PID(profile, pidFileName)

	daemon := Daemon{
		PidFileName: pidFileName,
		Profile:     profile,
		PidFilePath: pidFilePath,
	}

	return &daemon
}
