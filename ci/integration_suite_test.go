package ci_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NginxHostsReloadIntegration")
}
