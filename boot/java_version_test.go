package boot_test

import (
	"errors"
	"io"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/effect/mocks"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/spring-boot/v5/boot"
)

func testJavaMajorVersion(t *testing.T, context spec.G, it spec.S) {

	Expect := NewWithT(t).Expect
	var executor *mocks.Executor
	type testCase struct {
		name        string
		output      string
		major       int
		expectError bool
	}

	tests := []testCase{
		{
			name: "Temurin21",
			output: `openjdk version "21.0.5" 2024-10-15 LTS
OpenJDK Runtime Environment Temurin-21.0.5+11 (build 21.0.5+11-LTS)
OpenJDK 64-Bit Server VM Temurin-21.0.5+11 (build 21.0.5+11-LTS, mixed mode, sharing)`,
			major: 21,
		},
		{
			name: "Liberica25",
			output: `openjdk version "25" 2025-09-16 LTS
OpenJDK Runtime Environment Liberica-NIK-25.0.0-1 (build 25+37-LTS)
OpenJDK 64-Bit Server VM Liberica-NIK-25.0.0-1 (build 25+37-LTS, mixed mode, sharing)`,
			major: 25,
		},
		{
			name: "Java8",
			output: `java version "1.8.0_352"
Java(TM) SE Runtime Environment (build 1.8.0_352-b08)
Java HotSpot(TM) 64-Bit Server VM (build 25.352-b08, mixed mode)`,
			major: 8,
		},
		{
			name: "EarlyAccess",
			output: `openjdk version "17.0.1-ea" 2021-10-19
OpenJDK Runtime Environment (build 17.0.1-ea+12)
OpenJDK 64-Bit Server VM (build 17.0.1-ea+12, mixed mode, sharing)`,
			major: 17,
		},
		{
			name:   "Unparseable",
			output: `no version here`,
			major:  0, expectError: true,
		},
	}

	it("parses java -version output", func() {
		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				Expect := NewWithT(t).Expect

				major, err := boot.JavaMajorVersion(tc.output)

				if tc.expectError {
					Expect(err).To(HaveOccurred())
					return
				}

				Expect(err).NotTo(HaveOccurred())
				Expect(major).To(Equal(tc.major))
			})
		}
	})

	it("reads java version from executor output", func() {
		Expect(os.Setenv("JRE_HOME", "/that/does/not/exist")).To(Succeed())
		executor = &mocks.Executor{}
		executor.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			execution := args.Get(0).(effect.Execution)
			Expect(execution.Args).To(Equal([]string{"-version"}))
			if execution.Stderr != nil {
				_, err := io.WriteString(execution.Stderr, tests[0].output)
				Expect(err).NotTo(HaveOccurred())
			}
		}).Return(nil)

		major, err := boot.JavaMajorVersionFromJRE(executor)
		Expect(err).NotTo(HaveOccurred())
		Expect(major).To(Equal(21))
		Expect(os.Unsetenv("JRE_HOME")).To(Succeed())

	})

	it("returns error when executor fails", func() {
		executor = &mocks.Executor{}
		executor.On("Execute", mock.Anything).Return(errors.New("boom"))

		_, err := boot.JavaMajorVersionFromJRE(executor)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("boom"))
	})

	it("returns error when output empty", func() {
		executor = &mocks.Executor{}
		executor.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			execution := args.Get(0).(effect.Execution)
			Expect(execution.Args).To(Equal([]string{"-version"}))
			if execution.Stderr != nil {
				_, err := io.WriteString(execution.Stderr, "")
				Expect(err).NotTo(HaveOccurred())
			}
		}).Return(nil)

		_, err := boot.JavaMajorVersionFromJRE(executor)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("java -version produced no output"))
	})
}
