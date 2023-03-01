package install_ee

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tarantool/tt/cli/util"
	"github.com/tarantool/tt/cli/version"
)

type getVersionsInputValue struct {
	data *[]byte
}

type getVersionsOutputValue struct {
	result []EEVersion
	err    error
}

func TestGetVersions(t *testing.T) {
	assert := assert.New(t)

	testCases := make(map[getVersionsInputValue]getVersionsOutputValue)

	inputData0 := []byte("random string")

	testCases[getVersionsInputValue{data: &inputData0}] =
		getVersionsOutputValue{
			result: nil,
			err:    fmt.Errorf("no packages for this OS"),
		}

	arch, err := util.GetArch()
	assert.NoError(err)

	osType, err := util.GetOs()
	assert.NoError(err)
	inputData1 := []byte(``)
	osName := ""
	switch osType {
	case util.OsLinux:
		osName = ".linux."
	case util.OsMacos:
		osName = ".macos."
	}

	inputData1 = []byte(`/enterprise/tarantool-enterprise-sdk-` +
		`1.10.10-52-r419` + osName + arch +
		`.tar.gz`)

	testCases[getVersionsInputValue{data: &inputData1}] =
		getVersionsOutputValue{
			result: []EEVersion{
				EEVersion{VersionInfo: version.Version{
					Major:      1,
					Minor:      10,
					Patch:      10,
					Additional: 52,
					Revision:   419,
					Release:    version.Release{Type: version.TypeRelease},
					Hash:       "",
					Str:        "1.10.10-52-r419",
					Tarball: "tarantool-enterprise-sdk-1.10.10-52-r419" +
						osName + arch + ".tar.gz",
				}, Prefix: "/enterprise/"},
			},
			err: nil,
		}

	for input, output := range testCases {
		versions, err := getVersions(input.data)

		if output.err == nil {
			assert.Nil(err)
			assert.Equal(output.result, versions)
		} else {
			assert.Equal(output.err, err)
		}
	}
}

type getCredsFromFileInputValue struct {
	path string
}

type getCredsFromFileOutputValue struct {
	result userCredentials
	err    error
}

func TestGetCredsFromFile(t *testing.T) {
	assert := assert.New(t)

	testCases := make(map[getCredsFromFileInputValue]getCredsFromFileOutputValue)

	testCases[getCredsFromFileInputValue{path: "./testdata/nonexisting"}] =
		getCredsFromFileOutputValue{
			result: userCredentials{},
			err:    fmt.Errorf("open ./testdata/nonexisting: no such file or directory"),
		}

	testCases[getCredsFromFileInputValue{path: "./testdata/creds_ok"}] =
		getCredsFromFileOutputValue{
			result: userCredentials{
				username: "toor",
				password: "1234",
			},
			err: nil,
		}

	testCases[getCredsFromFileInputValue{path: "./testdata/creds_bad"}] =
		getCredsFromFileOutputValue{
			result: userCredentials{},
			err:    fmt.Errorf("corrupted credentials"),
		}

	for input, output := range testCases {
		creds, err := getCredsFromFile(input.path)

		if output.err == nil {
			assert.Nil(err)
			assert.Equal(output.result, creds)
		} else {
			assert.Equal(output.err.Error(), err.Error())
		}
	}
}

func Test_getVersionFromName(t *testing.T) {
	type args struct {
		bundleName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Plain test",
			args: args{
				bundleName: "testtest.test.test/test/asdsada1.10sdsad/",
			},
			want: "1.10",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getShortVersionFromBundleName(tt.args.bundleName)
			if !tt.wantErr(t, err, fmt.Sprintf("getVersionFromName(%v)", tt.args.bundleName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getVersionFromName(%v)", tt.args.bundleName)
		})
	}
}

func Test_getCredsFromEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		prepare func()
		want    userCredentials
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "Environment variables are not passed",
			prepare: func() {},
			want:    userCredentials{username: "", password: ""},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				if err.Error() == "no credentials in environment variables were found" {
					return true
				}
				return false
			},
		},
		{
			name: "Environment variables are passed",
			prepare: func() {
				t.Setenv("TT_EE_USERNAME", "tt_test")
				t.Setenv("TT_EE_PASSWORD", "tt_test")
			},
			want: userCredentials{username: "tt_test", password: "tt_test"},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			got, err := getCredsFromEnvVars()
			if !tt.wantErr(t, err, fmt.Sprintf("getCredsFromEnvVars()")) {
				return
			}
			assert.Equalf(t, tt.want, got, "getCredsFromEnvVars()")
		})
	}
}
