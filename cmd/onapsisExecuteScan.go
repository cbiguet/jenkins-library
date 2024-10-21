package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SAP/jenkins-library/pkg/command"
	piperHttp "github.com/SAP/jenkins-library/pkg/http"
	"github.com/SAP/jenkins-library/pkg/log"
	"github.com/SAP/jenkins-library/pkg/piperutils"
	"github.com/SAP/jenkins-library/pkg/telemetry"
	"github.com/pkg/errors"
)

type onapsisExecuteScanUtils interface {
	command.ExecRunner

	FileExists(filename string) (bool, error)
	Open(name string) (io.ReadWriteCloser, error)
	Getwd() (string, error)

	// Add more methods here, or embed additional interfaces, or remove/replace as required.
	// The onapsisExecuteScanUtils interface should be descriptive of your runtime dependencies,
	// i.e. include everything you need to be able to mock in tests.
	// Unit tests shall be executable in parallel (not depend on global state), and don't (re-)test dependencies.
}

type onapsisExecuteScanUtilsBundle struct {
	*command.Command
	*piperutils.Files

	// Embed more structs as necessary to implement methods or interfaces you add to onapsisExecuteScanUtils.
	// Structs embedded in this way must each have a unique set of methods attached.
	// If there is no struct which implements the method you need, attach the method to
	// onapsisExecuteScanUtilsBundle and forward to the implementation of the dependency.
}

func zipProject(folderPath string, outputPath string) error {
	// Create the output file
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	// Create a new zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk through all the files in the folder
	err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a header based on the file info
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Ensure the correct relative file path in the zip
		header.Name, err = filepath.Rel(filepath.Dir(folderPath), path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		// Create the writer for this file
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// If it's a file, copy the content into the zip
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to zip folder: %w", err)
	}

	return nil
}

func newOnapsisExecuteScanUtils() onapsisExecuteScanUtils {
	utils := onapsisExecuteScanUtilsBundle{
		Command: &command.Command{},
		Files:   &piperutils.Files{},
	}
	// Reroute command output to logging framework
	utils.Stdout(log.Writer())
	utils.Stderr(log.Writer())
	return &utils
}

func onapsisExecuteScan(config onapsisExecuteScanOptions, telemetryData *telemetry.CustomData) {
	// Utils can be used wherever the command.ExecRunner interface is expected.
	// It can also be used for example as a mavenExecRunner.
	utils := newOnapsisExecuteScanUtils()

	// For HTTP calls import  piperhttp "github.com/SAP/jenkins-library/pkg/http"
	// and use a  &piperhttp.Client{} in a custom system
	// Example: step checkmarxExecuteScan.go

	// Error situations should be bubbled up until they reach the line below which will then stop execution
	// through the log.Entry().Fatal() call leading to an os.Exit(1) in the end.
	err := runOnapsisExecuteScan(&config, telemetryData, utils)
	if err != nil {
		log.Entry().WithError(err).Fatal("step execution failed")
	}
}

func runOnapsisExecuteScan(config *onapsisExecuteScanOptions, telemetryData *telemetry.CustomData, utils onapsisExecuteScanUtils) error {
	log.Entry().WithField("LogField", "Log field content").Info("This is just a demo for a simple step.")

	// Create a new ScanServer
	server, err := NewScanServer(&piperHttp.Client{}, config.ScanServiceURL, config.AccessToken)
	if err != nil {
		return errors.Wrap(err, "failed to create scan server")
	}

	// Call the ScanProject method
	response, err := server.ScanProject(config, telemetryData, utils, "ui5")
	// Log the JobID
	log.Entry().Infof("JobID: %s", response.Result.JobID)

	// // Example of calling methods from external dependencies directly on utils:
	// exists, err := utils.FileExists("file.txt")
	// if err != nil {
	// 	// It is good practice to set an error category.
	// 	// Most likely you want to do this at the place where enough context is known.
	// 	log.SetErrorCategory(log.ErrorConfiguration)
	// 	// Always wrap non-descriptive errors to enrich them with context for when they appear in the log:
	// 	return fmt.Errorf("failed to check for important file: %w", err)
	// }
	// if !exists {
	// 	log.SetErrorCategory(log.ErrorConfiguration)
	// 	return fmt.Errorf("cannot run without important file")
	// }

	return nil
}

type ScanServer struct {
	serverUrl string
	client    piperHttp.Uploader
}

func NewScanServer(client piperHttp.Uploader, serverUrl string, token string) (*ScanServer, error) {
	server := &ScanServer{serverUrl: serverUrl, client: client}

	// Set authorization token for client
	options := piperHttp.ClientOptions{
		Token: "Bearer " + token,
	}
	server.client.SetOptions(options)

	return server, nil
}

func (srv *ScanServer) ScanProject(config *onapsisExecuteScanOptions, telemetryData *telemetry.CustomData, utils onapsisExecuteScanUtils, language string) (Response, error) {
	// // Zip workspace files
	// zipFile, err := zipProject(utils)
	// if err != nil {
	// 	return Response{}, errors.Wrap(err, "failed to zip workspace files")
	// }

	// // Get zip file content
	// file := zipFile.Name()
	// fileHandle, err := utils.Open(file)
	// if err != nil {
	// 	return Response{}, errors.Wrapf(err, "unable to locate file %v", file)
	// }
	// defer fileHandle.Close()

	// Get workspace path
	workspace, err := utils.Getwd()
	if err != nil {
		return Response{}, errors.Wrap(err, "failed to get workspace path")
	}
	zipFileName := filepath.Join(workspace, "workspace.zip")

	// Zip workspace files
	err = zipProject(workspace, zipFileName)
	if err != nil {
		return Response{}, errors.Wrap(err, "failed to zip workspace files")
	}

	// Get zip file content
	fileHandle, err := utils.Open(zipFileName)
	if err != nil {
		return Response{}, errors.Wrapf(err, "unable to locate file %v", zipFileName)
	}
	defer fileHandle.Close()

	// Construct ScanConfig form field
	scanConfig := fmt.Sprintf(`{
		"engine_type": "FILE",
		"scan_information": {
			"name": "scenario",
			"description": "a scan with extracted source"
		},
		"asset": {
			"file_format": "ZIP",
			"recursive": "true",
			"language": "%s"
		},
		"configuration": {},
		"scan_scope": {}
	}`, language)

	formFields := map[string]string{
		"ScanConfig": scanConfig,
	}

	// Create request data
	requestData := piperHttp.UploadRequestData{
		Method:        "POST",
		URL:           srv.serverUrl + "/cca/v1.0/scan/file",
		File:          zipFileName,
		FileFieldName: "FileUploadContent",
		FileContent:   fileHandle,
		FormFields:    formFields,
		UploadType:    "form",
	}

	// Send request
	response, err := srv.client.Upload(requestData)
	if err != nil {
		return Response{}, errors.Wrap(err, "failed to upload file")
	}

	// Parse response
	responseData := Response{}
	err = piperHttp.ParseHTTPResponseBodyJSON(response, responseData)
	if err != nil {
		return Response{}, errors.Wrap(err, "failed to parse file")
	}

	// Check the success field
	if responseData.Success {
		return responseData, nil
	} else {
		return responseData, errors.Errorf("Request failed with result_code: %d, messages: %v", responseData.Result.ResultCode, responseData.Result.Messages)
	}

}

// func zipWorkspace(utils onapsisExecuteScanUtils) (*os.File, error) {
// 	zipFileName := filepath.Join(utils.GetWorkspace(), "workspace.zip")
// 	patterns := piperutils.Trim(strings.Split(filterPattern, ","))
// 	sort.Strings(patterns)
// 	zipFile, err := os.Create(zipFileName)
// 	if err != nil {
// 		return zipFile, errors.Wrap(err, "failed to create archive of project sources")
// 	}
// 	defer zipFile.Close()
// 	err = zipFolder(utils.GetWorkspace(), zipFile, patterns, utils)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "failed to compact folder")
// 	}
// 	return zipFile, nil
// }

type Response struct {
	Success bool             `json:"success"`
	Result  OnapsisJobResult `json:"result"`
}

type OnapsisJobResult struct {
	JobID      string    `json:"job_id,omitempty"`      // present only on success
	ResultCode int       `json:"result_code,omitempty"` // present only on failure
	Timestamp  string    `json:"timestamp,omitempty"`   // present only on success
	Messages   []Message `json:"messages"`
}

type Message struct {
	Sequence  int     `json:"sequence"`
	Timestamp string  `json:"timestamp"`
	Level     string  `json:"level"`
	MessageID string  `json:"message_id"`
	Param1    *string `json:"param1"`
	Param2    *string `json:"param2"`
	Param3    *string `json:"param3"`
	Param4    *string `json:"param4"`
}
