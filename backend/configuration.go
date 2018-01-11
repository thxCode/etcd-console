package backend

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/kataras/iris"
	"gopkg.in/yaml.v2"
	"github.com/kataras/iris/core/errors"
)

const globalConfigurationKeyword = "~"

var errConfigurationDecode = errors.New("error while trying to decode configuration")

// homeConfigurationFilename returns the physical location of the global configuration(yaml or toml) file.
// This is useful when we run multiple iris servers that share the same
// configuration, even with custom values at its "Other" field.
// It will return a file location
// which targets to $HOME or %HOMEDRIVE%+%HOMEPATH% + "iris" + the given "ext".
func homeConfigurationFilename(ext string) string {
	return filepath.Join(homeDir(), "iris"+ext)
}

func homeDir() (home string) {
	u, err := user.Current()
	if u != nil && err == nil {
		home = u.HomeDir
	}

	if home == "" {
		home = os.Getenv("HOME")
	}

	if home == "" {
		if runtime.GOOS == "plan9" {
			home = os.Getenv("home")
		} else if runtime.GOOS == "windows" {
			home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
			if home == "" {
				home = os.Getenv("USERPROFILE")
			}
		}
	}

	return
}

func parseYAML(filename string) (Configuration, error) {
	c := DefaultConfiguration()
	// get the abs
	// which will try to find the 'filename' from current workind dir too.
	yamlAbsPath, err := filepath.Abs(filename)
	if err != nil {
		return c, errConfigurationDecode.AppendErr(err)
	}

	// read the raw contents of the file
	data, err := ioutil.ReadFile(yamlAbsPath)
	if err != nil {
		return c, errConfigurationDecode.AppendErr(err)
	}

	// put the file's contents as yaml to the default configuration(c)
	if err := yaml.Unmarshal(data, &c); err != nil {
		return c, errConfigurationDecode.AppendErr(err)
	}
	return c, nil
}

func YAML(filename string) Configuration {
	// check for globe configuration file and use that, otherwise
	// return the default configuration if file doesn't exist.
	if filename == globalConfigurationKeyword {
		filename = homeConfigurationFilename(".yml")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			panic("default configuration file '" + filename + "' does not exist")
		}
	}

	c, err := parseYAML(filename)
	if err != nil {
		panic(err)
	}

	if c.Endpoints == nil && len(c.Endpoints) == 0 {
		panic("cannot get etcd endpoints.")
	}

	return c
}

type Configuration struct {
	// The address is used for communicating etcd-console data.
	// Defaults to ":8080".
	Advertise string `json:"advertise,omitempty" yaml:"Advertise"`

	// Specify using endpoints of etcd, splitting by comma.
	// Defaults to "http://127.0.0.1:2379"
	Endpoints []string `json:"endpoints,omitempty" yaml:"Endpoints"`

	// Start with an embedding etcd or not.
	// Defaults to "true"
	Test bool `json:"test,omitempty" yaml:"Test"`

	// Start with an embedding etcd or not.
	// Defaults to "true"
	LogLevel string `json:"logLevel,omitempty" yaml:"LogLevel"`

	// Where is storing the backup zip files.
	// Defaults to "/tmp/etcd_console.backup"
	BackupDir string `json:"backupDir,omitempty"`

	////////////////////////
	// iris.Configuration //
	///////////////////////

	// IgnoreServerErrors will cause to ignore the matched "errors"
	// from the main application's `Run` function.
	// This is a slice of string, not a slice of error
	// users can register these errors using yaml or toml configuration file
	// like the rest of the configuration fields.
	//
	// See `WithoutServerError(...)` function too.
	//
	// Example: https://github.com/kataras/iris/tree/v8/_examples/http-listening/listen-addr/omit-server-errors
	//
	// Defaults to an empty slice.
	IgnoreServerErrors []string `json:"ignoreServerErrors,omitempty" yaml:"IgnoreServerErrors" toml:"IgnoreServerErrors"`

	// DisableStartupLog if setted to true then it turns off the write banner on server startup.
	//
	// Defaults to false.
	DisableStartupLog bool `json:"disableStartupLog,omitempty" yaml:"DisableStartupLog" toml:"DisableStartupLog"`
	// DisableInterruptHandler if setted to true then it disables the automatic graceful server shutdown
	// when control/cmd+C pressed.
	// Turn this to true if you're planning to handle this by your own via a custom host.Task.
	//
	// Defaults to false.
	DisableInterruptHandler bool `json:"disableInterruptHandler,omitempty" yaml:"DisableInterruptHandler" toml:"DisableInterruptHandler"`

	// DisableVersionChecker if true then process will be not be notified for any available updates.
	//
	// Defaults to false.
	DisableVersionChecker bool `json:"disableVersionChecker,omitempty" yaml:"DisableVersionChecker" toml:"DisableVersionChecker"`

	// DisablePathCorrection corrects and redirects the requested path to the registered path
	// for example, if /home/ path is requested but no handler for this Route found,
	// then the Router checks if /home handler exists, if yes,
	// (permant)redirects the client to the correct path /home
	//
	// Defaults to false.
	DisablePathCorrection bool `json:"disablePathCorrection,omitempty" yaml:"DisablePathCorrection" toml:"DisablePathCorrection"`

	// EnablePathEscape when is true then its escapes the path, the named parameters (if any).
	// Change to false it if you want something like this https://github.com/kataras/iris/issues/135 to work
	//
	// When do you need to Disable(false) it:
	// accepts parameters with slash '/'
	// Request: http://localhost:8080/details/Project%2FDelta
	// ctx.Param("project") returns the raw named parameter: Project%2FDelta
	// which you can escape it manually with net/url:
	// projectName, _ := url.QueryUnescape(c.Param("project").
	//
	// Defaults to false.
	EnablePathEscape bool `json:"enablePathEscape,omitempty" yaml:"EnablePathEscape" toml:"EnablePathEscape"`

	// EnableOptimization when this field is true
	// then the application tries to optimize for the best performance where is possible.
	//
	// Defaults to false.
	EnableOptimizations bool `json:"enableOptimizations,omitempty" yaml:"EnableOptimizations" toml:"EnableOptimizations"`
	// FireMethodNotAllowed if it's true router checks for StatusMethodNotAllowed(405) and
	//  fires the 405 error instead of 404
	// Defaults to false.
	FireMethodNotAllowed bool `json:"fireMethodNotAllowed,omitempty" yaml:"FireMethodNotAllowed" toml:"FireMethodNotAllowed"`

	// DisableBodyConsumptionOnUnmarshal manages the reading behavior of the context's body readers/binders.
	// If setted to true then it
	// disables the body consumption by the `context.UnmarshalBody/ReadJSON/ReadXML`.
	//
	// By-default io.ReadAll` is used to read the body from the `context.Request.Body which is an `io.ReadCloser`,
	// if this field setted to true then a new buffer will be created to read from and the request body.
	// The body will not be changed and existing data before the
	// context.UnmarshalBody/ReadJSON/ReadXML will be not consumed.
	DisableBodyConsumptionOnUnmarshal bool `json:"disableBodyConsumptionOnUnmarshal,omitempty" yaml:"DisableBodyConsumptionOnUnmarshal" toml:"DisableBodyConsumptionOnUnmarshal"`

	// DisableAutoFireStatusCode if true then it turns off the http error status code handler automatic execution
	// from "context.StatusCode(>=400)" and instead app should manually call the "context.FireStatusCode(>=400)".
	//
	// By-default a custom http error handler will be fired when "context.StatusCode(code)" called,
	// code should be >=400 in order to be received as an "http error handler".
	//
	// Developer may want this option to setted as true in order to manually call the
	// error handlers when needed via "context.FireStatusCode(>=400)".
	// HTTP Custom error handlers are being registered via app.OnErrorCode(code, handler)".
	//
	// Defaults to false.
	DisableAutoFireStatusCode bool `json:"disableAutoFireStatusCode,omitempty" yaml:"DisableAutoFireStatusCode" toml:"DisableAutoFireStatusCode"`

	// TimeFormat time format for any kind of datetime parsing
	// Defaults to  "Mon, 02 Jan 2006 15:04:05 GMT".
	TimeFormat string `json:"timeFormat,omitempty" yaml:"TimeFormat" toml:"TimeFormat"`

	// Charset character encoding for various rendering
	// used for templates and the rest of the responses
	// Defaults to "UTF-8".
	Charset string `json:"charset,omitempty" yaml:"Charset" toml:"Charset"`

	//  +----------------------------------------------------+
	//  | Context's keys for values used on various featuers |
	//  +----------------------------------------------------+

	// Context values' keys for various features.
	//
	// TranslateLanguageContextKey & TranslateFunctionContextKey are used by i18n handlers/middleware
	// currently we have only one: https://github.com/kataras/iris/tree/v8/middleware/i18n.
	//
	// Defaults to "iris.translate" and "iris.language"
	TranslateFunctionContextKey string `json:"translateFunctionContextKey,omitempty" yaml:"TranslateFunctionContextKey" toml:"TranslateFunctionContextKey"`
	// TranslateLanguageContextKey used for i18n.
	//
	// Defaults to "iris.language"
	TranslateLanguageContextKey string `json:"translateLanguageContextKey,omitempty" yaml:"TranslateLanguageContextKey" toml:"TranslateLanguageContextKey"`

	// GetViewLayoutContextKey is the key of the context's user values' key
	// which is being used to set the template
	// layout from a middleware or the main handler.
	// Overrides the parent's or the configuration's.
	//
	// Defaults to "iris.ViewLayout"
	ViewLayoutContextKey string `json:"viewLayoutContextKey,omitempty" yaml:"ViewLayoutContextKey" toml:"ViewLayoutContextKey"`
	// GetViewDataContextKey is the key of the context's user values' key
	// which is being used to set the template
	// binding data from a middleware or the main handler.
	//
	// Defaults to "iris.viewData"
	ViewDataContextKey string `json:"viewDataContextKey,omitempty" yaml:"ViewDataContextKey" toml:"ViewDataContextKey"`
	// RemoteAddrHeaders returns the allowed request headers names
	// that can be valid to parse the client's IP based on.
	//
	// Defaults to:
	// "X-Real-Ip":             false,
	// "X-Forwarded-For":       false,
	// "CF-Connecting-IP": false
	//
	// Look `context.RemoteAddr()` for more.
	RemoteAddrHeaders map[string]bool `json:"remoteAddrHeaders,omitempty" yaml:"RemoteAddrHeaders" toml:"RemoteAddrHeaders"`

	// Other are the custom, dynamic options, can be empty.
	// This field used only by you to set any app's options you want
	// or by custom adaptors, it's a way to simple communicate between your adaptors (if any)
	// Defaults to a non-nil empty map.
	Other map[string]interface{} `json:"other,omitempty" yaml:"Other" toml:"Other"`
}

func DefaultConfiguration() Configuration {
	return Configuration{
		Advertise: ":8080",
		Endpoints: []string{"http://127.0.0.1:2379"},
		Test:      true,
		LogLevel:  "debug",

		///////////////////////////////
		// iris.DefaultConfiguration //
		///////////////////////////////

		DisableStartupLog:                 false,
		DisableInterruptHandler:           false,
		DisableVersionChecker:             false,
		DisablePathCorrection:             false,
		EnablePathEscape:                  false,
		FireMethodNotAllowed:              false,
		DisableBodyConsumptionOnUnmarshal: false,
		DisableAutoFireStatusCode:         false,
		TimeFormat:                        "Mon, Jan 02 2006 15:04:05 GMT",
		Charset:                           "UTF-8",
		TranslateFunctionContextKey:       "iris.translate",
		TranslateLanguageContextKey:       "iris.language",
		ViewLayoutContextKey:              "iris.viewLayout",
		ViewDataContextKey:                "iris.viewData",
		RemoteAddrHeaders: map[string]bool{
			"X-Real-Ip":        false,
			"X-Forwarded-For":  false,
			"CF-Connecting-IP": false,
		},
		EnableOptimizations: false,
		Other:               make(map[string]interface{}),
	}
}

func (c Configuration) ToIrisConfiguration() iris.Configuration {
	return iris.Configuration{
		IgnoreServerErrors:                c.IgnoreServerErrors,
		DisableStartupLog:                 c.DisableStartupLog,
		DisableInterruptHandler:           c.DisableInterruptHandler,
		DisableVersionChecker:             c.DisableVersionChecker,
		DisablePathCorrection:             c.DisablePathCorrection,
		EnablePathEscape:                  c.EnablePathEscape,
		EnableOptimizations:               c.EnableOptimizations,
		FireMethodNotAllowed:              c.FireMethodNotAllowed,
		DisableBodyConsumptionOnUnmarshal: c.DisableBodyConsumptionOnUnmarshal,
		DisableAutoFireStatusCode:         c.DisableAutoFireStatusCode,
		TimeFormat:                        c.TimeFormat,
		Charset:                           c.Charset,
		TranslateFunctionContextKey:       c.TranslateFunctionContextKey,
		TranslateLanguageContextKey:       c.TranslateLanguageContextKey,
		ViewLayoutContextKey:              c.ViewLayoutContextKey,
		ViewDataContextKey:                c.ViewDataContextKey,
		RemoteAddrHeaders:                 c.RemoteAddrHeaders,
		Other:                             c.Other,
	}
}
