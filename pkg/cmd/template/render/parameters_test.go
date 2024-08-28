package render

import (
	"bytes"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/coreeng/corectl/testdata"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("template arguments are collected", func() {
	var (
		templateSpec *template.Spec
		t            FullGinkgoTInterface
	)

	BeforeEach(func() {
		var err error
		templateSpec, err = template.FindByName(
			testdata.TemplatesPath(),
			testdata.TemplateWithArgs(),
		)
		Expect(err).NotTo(HaveOccurred())
		t = GinkgoT()
	})

	It("from args file", func() {
		argsFile := createArgsFile(t.TempDir(), map[string]string{
			"param1":    "value",
			"param2":    "123",
			"from-file": "value",
		})
		args, err := parseArgsFile(
			templateSpec,
			argsFile,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(args).To(ConsistOf([]template.Argument{
			{
				Name:  "param1",
				Value: "value",
			},
			{
				Name:  "param2",
				Value: 123,
			},
		}))
	})

	Context("from flags", func() {
		It("success", func() {
			args, err := parseArgsFromFlags(
				templateSpec,
				[]string{"param1=abc", "param2=123", "from-flags=value"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(Equal([]template.Argument{
				{
					Name:  "param1",
					Value: "abc",
				},
				{
					Name:  "param2",
					Value: 123,
				},
			}))
		})
		It("invalid param", func() {
			args, err := parseArgsFromFlags(
				templateSpec,
				[]string{"param1=abc", "param2"},
			)
			Expect(args).To(BeNil())
			Expect(err).To(MatchError(ContainSubstring("expected format: <arg-name>=<arg-value>")))
		})
		It("invalid param type", func() {
			args, err := parseArgsFromFlags(
				templateSpec,
				[]string{"param1=abc", "param2=cba"},
			)
			Expect(args).To(BeNil())
			Expect(err).To(MatchError(ContainSubstring("invalid param2 arg value")))
		})
	})

	Context("from all sources", func() {
		It("with non-interactive input => skips optional args", func(ctx SpecContext) {
			argsFile := createArgsFile(t.TempDir(), map[string]string{
				"name":   "app-name",
				"tenant": "tenant-name",
			})
			stdin, stdout := bytes.Buffer{}, bytes.Buffer{}
			streams := userio.NewTestIOStreams(
				&stdin,
				&stdout,
				false,
			)

			args, err := CollectArgsFromAllSources(
				templateSpec,
				argsFile,
				[]string{"from-flag=value", "param1=value"},
				streams,
				[]template.Argument{
					{Name: "param2", Value: 123},
					{Name: "fixed-arg", Value: "value"},
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(ConsistOf([]template.Argument{
				// From file
				{
					Name:  "name",
					Value: "app-name",
				},
				{
					Name:  "tenant",
					Value: "tenant-name",
				},
				// From flags
				{
					Name:  "param1",
					Value: "value",
				},
				// Fixed args
				{
					Name:  "param2",
					Value: 123,
				},
				// Default args
				{
					Name:  "param4",
					Value: 9876,
				},
				// From input: no args, skipped optional
			}))
		}, NodeTimeout(time.Second*10))

		It("with interactive input", func(ctx SpecContext) {
			argsFile := createArgsFile(t.TempDir(), map[string]string{
				"name":   "app-name",
				"tenant": "tenant-name",
				"param4": "3456",
			})
			stdin, stdout := bytes.Buffer{}, bytes.Buffer{}
			streams := userio.NewTestIOStreams(
				&stdin,
				&stdout,
				true,
			)

			// param1
			stdin.WriteString("value")
			stdin.WriteByte(byte(tea.KeyEnter))

			args, err := CollectArgsFromAllSources(
				templateSpec,
				argsFile,
				[]string{"from-flag=value", "param2=123"},
				streams,
				[]template.Argument{
					{Name: "fixed-arg", Value: "value"},
					{Name: "param3", Value: "value"},
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(args).To(ConsistOf([]template.Argument{
				// From file
				{
					Name:  "name",
					Value: "app-name",
				},
				{
					Name:  "tenant",
					Value: "tenant-name",
				},
				{
					Name:  "param4",
					Value: 3456,
				},
				// From flags
				{
					Name:  "param2",
					Value: 123,
				},
				// Fixed args
				{
					Name:  "param3",
					Value: "value",
				},
				// From input
				{
					Name:  "param1",
					Value: "value",
				},
			}))
		}, NodeTimeout(time.Second*10))
	})
})
