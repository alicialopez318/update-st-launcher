/**
 * Copyright (c) 2021 BlockDev AG
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package native

import (
	"log"
	"time"

	model_ "github.com/mysteriumnetwork/myst-launcher/model"
	"github.com/mysteriumnetwork/myst-launcher/utils"
)

type Controller struct {
	a model_.AppState

	finished bool
	runner   *NodeRunner
	lg       *log.Logger
}

func (c *Controller) GetFinished() bool {
	return c.finished
}

func (c *Controller) SetFinished() {
	c.finished = true
}

func NewController() *Controller {
	lg := log.New(log.Writer(), "[native] ", log.Ldate|log.Ltime)
	return &Controller{lg: lg}
}

func (c *Controller) GetCaps() int {
	return 0
}

func (c *Controller) SetApp(a model_.AppState) {
	c.a = a
	c.runner = NewRunner(a.GetModel())
}

func (c *Controller) Shutdown() {}

// Supervise the node
func (c *Controller) Start() {
	defer utils.PanicHandler("app-2")
	c.lg.Println("start")

	model := c.a.GetModel()
	action := c.a.GetAction()
	cfg := model.Config

	// copy version info to ui model
	model.ImageInfo.VersionCurrent = cfg.NodeExeVersion
	model.ImageInfo.VersionLatest = cfg.NodeLatestTag
	model.Update()

	defer c.SetFinished()

	t1 := time.NewTicker(15 * time.Second)
	for {
		model.SwitchState(model_.UIStateInitial)

		c.startContainer()
		c.upgradeContainer(false)
		// if model.Config.AutoUpgrade {
		// c.upgradeContainer(false)
		// }

		// c.lg.Println("wait action >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		select {
		case act := <-action:
			c.lg.Println("action:", act)

			switch act {
			case model_.ActionCheck:
				c.upgradeContainer(true)

			case model_.ActionUpgrade:
				c.upgradeContainer(false)

			case model_.ActionRestart:
				// restart to apply new settings
				c.restartContainer()
				model.Config.Save()

			case model_.ActionEnable:
				model.SetStateContainer(model_.RunnableStateStarting)
				c.startContainer()
				model.SetStateContainer(model_.RunnableStateRunning)

			case model_.ActionDisable:
				model.SetStateContainer(model_.RunnableStateUnknown)
				c.stop()

			case model_.ActionStopRunner:
				// terminate controller
				model.SetStateContainer(model_.RunnableStateUnknown)
				c.stop()
				return

			case model_.ActionStop:
				c.lg.Println("[native] stop")
				return
			}

		// wait for ticker event if no action
		case <-t1.C:
		}
	}
}

func (c *Controller) restartContainer() {
	model := c.a.GetModel()
	model.SetStateContainer(model_.RunnableStateInstalling)

	c.runner.Stop()
	c.runner.IsRunningOrTryStart()
	model.SetStateContainer(model_.RunnableStateRunning)
}

func (c *Controller) stop() {
	c.runner.Stop()
}

func (c *Controller) upgradeContainer(refreshVersionCache bool) {
	model := c.a.GetModel()

	// if !model.ImageInfo.HasUpdate {
	// 	return
	// }

	model.SetStateContainer(model_.RunnableStateInstalling)
	c.CheckAndUpgradeNodeExe(refreshVersionCache)
	model.SetStateContainer(model_.RunnableStateRunning)
}

// check for image updates before starting container, offer upgrade interactively
func (c *Controller) startContainer() {
	c.lg.Println("!run")
	model := c.a.GetModel()

	model.SetStateContainer(model_.RunnableStateInstalling)
	if model.Config.Enabled {

		ui := c.a.GetUI()
		tryInstallFirewallRules(ui)
		
		running := c.runner.IsRunningOrTryStart()
		if running {
			model.SetStateContainer(model_.RunnableStateRunning)
		}
	}
}
