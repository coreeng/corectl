package git

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/coreeng/corectl/pkg/testutil/httpmock"
	"github.com/google/go-github/v59/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/phuslu/log"
)

func TestUpdate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auto Update Suite")
}

var _ = Describe("corectl update", func() {
	var (
		latestRelease             *github.RepositoryRelease
		specificRelease           *github.RepositoryRelease
		latestReleaseTag          string
		specificReleaseTag        string
		githubClient              *github.Client
		githubErrorClient         *github.Client
		getLatestReleaseCapture   *httpmock.HttpCaptureHandler[github.RepositoryRelease]
		getSpecificReleaseCapture *httpmock.HttpCaptureHandler[github.RepositoryRelease]
		githubErrorString         string
	)

	BeforeEach(OncePerOrdered, func() {
		log.DefaultLogger.SetLevel(log.PanicLevel)
		githubErrorString = "api error"
		latestReleaseTag = "v100.0.0"
		specificReleaseTag = "v0.0.1"
		latestRelease = &github.RepositoryRelease{TagName: github.String(latestReleaseTag)}
		specificRelease = &github.RepositoryRelease{TagName: github.String(specificReleaseTag)}
		getLatestReleaseCapture = httpmock.NewCaptureHandler[github.RepositoryRelease](latestRelease)
		getSpecificReleaseCapture = httpmock.NewCaptureHandler[github.RepositoryRelease](specificRelease)

		githubClient = github.NewClient(mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposReleasesLatestByOwnerByRepo,
				getLatestReleaseCapture.Func(),
			),
			mock.WithRequestMatchHandler(
				mock.GetReposReleasesTagsByOwnerByRepoByTag,
				getSpecificReleaseCapture.Func(),
			),
		))

		errorResponse := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mock.WriteError(
				w,
				http.StatusInternalServerError,
				githubErrorString,
			)
		})

		githubErrorClient = github.NewClient(mock.NewMockedHTTPClient(
			mock.WithRequestMatchHandler(
				mock.GetReposReleasesLatestByOwnerByRepo,
				errorResponse,
			),
			mock.WithRequestMatchHandler(
				mock.GetReposReleasesTagsByOwnerByRepoByTag,
				errorResponse,
			),
		))
	})

	Context("git.GetLatestCorectlRelease", Ordered, func() {
		It("returns the latest release", func() {
			release, err := GetLatestCorectlRelease(githubClient)
			Expect(release).Should(Equal(latestRelease))
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("returns an error when the API call fails", func() {
			_, err := GetLatestCorectlRelease(githubErrorClient)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring(githubErrorString))
		})

	})

	Context("git.GetCorectlReleaseByTag", Ordered, func() {
		It("returns the release for a specific tag", func() {
			release, err := GetCorectlReleaseByTag(githubClient, specificReleaseTag)
			Expect(release).Should(Equal(specificRelease))
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("returns an error when the API call fails", func() {
			release, err := GetCorectlReleaseByTag(githubErrorClient, specificReleaseTag)
			Expect(release).Should(Equal(&github.RepositoryRelease{}))
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring(githubErrorString))
		})
	})

	Context("git.DecompressCorectlAssetInMemory", func() {
		It("successfully decompresses and finds corectl binary", func() {
			// Create a mock gzipped tar archive containing a corectl binary
			mockTarGz := createMockTarGz("corectl", []byte("mock binary content"))
			reader, err := DecompressCorectlAssetInMemory(io.NopCloser(mockTarGz))

			Expect(err).ShouldNot(HaveOccurred())
			Expect(reader).ShouldNot(BeNil())

			content, err := io.ReadAll(reader)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(content)).Should(Equal("mock binary content"))
		})

		It("returns an error when corectl binary is not found", func() {
			mockTarGz := createMockTarGz("not-corectl", []byte("wrong content"))
			_, err := DecompressCorectlAssetInMemory(io.NopCloser(mockTarGz))

			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("corectl binary not found in the release"))
		})
	})

	Context("git.WriteCorectlAssetToPath", func() {
		var tmpFile *os.File
		var tmpPath string

		BeforeEach(func() {
			var err error
			tmpFile, err = os.CreateTemp("", "corectl-test")
			Expect(err).ShouldNot(HaveOccurred())

			tmpPath, err = os.Readlink(fmt.Sprintf("/proc/self/fd/%d", tmpFile.Fd()))

			Expect(err).ShouldNot(HaveOccurred())
		})

		It("successfully writes corectl binary to the specified path", func() {
			defer tmpFile.Close()
			mockTarGz := createMockTarGz("corectl", []byte("mock binary content"))
			gzipReader, err := gzip.NewReader(bytes.NewReader(mockTarGz.Bytes()))
			Expect(err).ShouldNot(HaveOccurred())
			mockTarReader := tar.NewReader(gzipReader)
			_, err = mockTarReader.Next() // set cursor to where it would be after iteration
			Expect(err).ShouldNot(HaveOccurred())

			err = WriteCorectlAssetToPath(mockTarReader, tmpPath, tmpFile)

			Expect(err).ShouldNot(HaveOccurred())

			writtenContent, err := os.ReadFile(tmpPath)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(writtenContent)).Should(Equal("mock binary content"))

			info, err := os.Stat(tmpPath)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(info.Mode().Perm()).Should(Equal(os.FileMode(0755)))
		})

		It("returns an error when writing fails", func() {
			defer tmpFile.Close()
			mockReader := strings.NewReader("mock binary content")
			tmpPath = "/non-existent-dir/corectl"

			err := WriteCorectlAssetToPath(tar.NewReader(mockReader), tmpPath, tmpFile)

			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("no such file or directory"))
		})
	})
})

func createMockTarGz(filename string, content []byte) *bytes.Buffer {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	hdr := &tar.Header{
		Name: filename,
		Mode: 0600,
		Size: int64(len(content)),
	}
	err := tw.WriteHeader(hdr)
	Expect(err).ShouldNot(HaveOccurred())

	_, err = tw.Write(content)
	Expect(err).ShouldNot(HaveOccurred())

	tw.Close()
	gzw.Close()

	return &buf
}