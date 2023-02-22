package main_test

import (
	"io/ioutil"
	"log"
	"time"

	"code.cloudfoundry.org/lager"
	. "github.com/alphagov/paas-nginx-hosts-reload"
	"github.com/alphagov/paas-nginx-hosts-reload/utils/utilsfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NginxHostsReload", func() {

	var hm *NginxHostsReload
	var currentModTime time.Time
	var currentPID int32
	var utils *utilsfakes.FakeUtils

	BeforeEach(func() {
		log.SetOutput(ioutil.Discard)
		utils = &utilsfakes.FakeUtils{}
		hm = NewNginxHostsReload(100*time.Millisecond, lager.NewLogger("test"), utils, "nginx", "master process")
		currentModTime = time.Now()
		currentPID = 0

		utils.FindNginxPIDStub = func(string, string) (int32, error) {
			return currentPID, nil
		}
		utils.SigHupReturns(nil)
		utils.ModTimeStub = func() (time.Time, error) {
			return currentModTime, nil
		}
	})

	Context("When the hosts file is modified", func() {
		It("Should send SIGHUP to nginx", func() {
			go func() {
				hm.Monitor()
			}()
			currentPID = 100
			time.Sleep(10 * time.Millisecond)
			currentModTime = time.Now()
			time.Sleep(1000 * time.Millisecond)
			Expect(utils.FindNginxPIDCallCount()).To(Equal(1))
			process, cmdSubstring := utils.FindNginxPIDArgsForCall(0)
			Expect(process).To(Equal("nginx"))
			Expect(cmdSubstring).To(Equal("master process"))
			Expect(utils.SigHupCallCount()).To(Equal(1))
			Expect(utils.SigHupArgsForCall(0)).To(Equal(int32(100)))
			hm.Stop()
		})
	})

	Context("When the hosts file is not modified", func() {
		It("Should not send SIGHUP to nginx", func() {
			go func() {
				hm.Monitor()
			}()
			time.Sleep(200 * time.Millisecond)
			Expect(utils.SigHupCallCount()).To(Equal(0))
			hm.Stop()
		})
	})

	Context("When the nginx process is not found", func() {
		It("Should not send SIGHUP to nginx", func() {
			currentPID = 0
			go func() {
				hm.Monitor()
			}()
			currentModTime = time.Now()
			time.Sleep(200 * time.Millisecond)
			Expect(utils.SigHupCallCount()).To(Equal(0))

			hm.Stop()
		})
	})

})
