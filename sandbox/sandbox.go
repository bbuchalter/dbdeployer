// DBDeployer - The MySQL Sandbox
// Copyright © 2006-2018 Giuseppe Maxia
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/datacharmer/dbdeployer/common"
	"github.com/datacharmer/dbdeployer/concurrent"
	"github.com/datacharmer/dbdeployer/defaults"
)

type SandboxDef struct {
	DirName              string           // Name of the directory containing the sandbox
	SBType               string           // Type of sandbox (single, multiple, replication-node, group-node)
	Multi                bool             // Either single or part of a multiple sandbox
	NodeNum              int              // In multiple sandboxes, which node is this
	Version              string           // MySQL version
	Basedir              string           // Where to get binaries from (e.g. $HOME/opt/mysql/8.0.11)
	BasedirName          string           // The bare name of the directory containing the binaries (e.g. 8.0.11)
	SandboxDir           string           // Target directory for sandboxes
	LoadGrants           bool             // Should we load grants?
	SkipReportHost       bool             // Do not add report-host to my.sandbox.cnf
	SkipReportPort       bool             // Do not add report-port to my.sandbox.cnf
	SkipStart            bool             // Do not start the server after deployment
	InstalledPorts       []int            // Which ports should be skipped in port assignment for this SB
	Port                 int              // Port assigned to this sandbox
	MysqlXPort           int              // XPlugin port for this sandbox
	UserPort             int              // Custom port provided by user
	BasePort             int              // Base port for calculating more ports in multiple SB
	MorePorts            []int            // Additional ports that belong to thos sandbox
	Prompt               string           // Prompt to use in "mysql" client
	DbUser               string           // Database user name
	RplUser              string           // Replication user name
	DbPassword           string           // Database password
	RplPassword          string           // Replication password
	RemoteAccess         string           // What access have the users created for this SB (127.%)
	BindAddress          string           // Bind address for this sandbox (127.0.0.1)
	CustomMysqld         string           // Use an alternative mysqld executable
	ServerId             int              // Server ID (for replication)
	ReplOptions          string           // Replication options, as string to append to my.sandbox.cnf
	GtidOptions          string           // Options needed for GTID
	ReplCrashSafeOptions string           // Options needed for Replication crash safe
	SemiSyncOptions      string           // Options for semi-synchronous replication
	InitOptions          []string         // Options to be added to the initialization command
	MyCnfOptions         []string         // Options to be added to my.sandbox.cnf
	PreGrantsSql         []string         // SQL statements to execute before grants assignment
	PreGrantsSqlFile     string           // SQL file to load before grants assignment
	PostGrantsSql        []string         // SQL statements to run after grants assignment
	PostGrantsSqlFile    string           // SQL file to load after grants assignment
	MyCnfFile            string           // options file to merge with the SB my.sandbox.cnf
	HistoryDir           string           // Where to store the MySQL client history
	LogFileName          string           // Where to log operations for this sandbox
	Logger               *defaults.Logger // Carries a logger across sandboxes
	InitGeneralLog       bool             // Enable general log during server initialization
	EnableGeneralLog     bool             // Enable general log for regular usage
	NativeAuthPlugin     bool             // Use the native password plugin for MySQL 8.0.4+
	DisableMysqlX        bool             // Disable Xplugin (MySQL 8.0.11+)
	EnableMysqlX         bool             // Enable Xplugin (MySQL 5.7.12+)
	KeepUuid             bool             // Do not change UUID
	SinglePrimary        bool             // Use single primary for group replication
	Force                bool             // Overwrite an existing sandbox with same target
	ExposeDdTables       bool             // Show hidden data dictionary tables (MySQL 8.0.0+)
	RunConcurrently      bool             // Run multiple sandbox creation concurrently
}

func GetOptionsFromFile(filename string) (options []string) {
	skip_options := map[string]bool{
		"user":         true,
		"port":         true,
		"socket":       true,
		"datadir":      true,
		"basedir":      true,
		"tmpdir":       true,
		"pid-file":     true,
		"server-id":    true,
		"bind-address": true,
		"log-error":    true,
	}
	config := common.ParseConfigFile(filename)
	for _, kv := range config["mysqld"] {

		if skip_options[kv.Key] {
			continue
		}
		options = append(options, fmt.Sprintf("%s = %s", kv.Key, kv.Value))
		//fmt.Printf("%d %s : %s \n", N, kv.key, kv.value)
	}
	return options
}

func SandboxDefToJson(sd SandboxDef) string {
	b, err := json.MarshalIndent(sd, " ", "\t")
	common.ErrCheckExitf(err, 1, "error encoding sandbox definition: %s", err)
	return fmt.Sprintf("%s", b)
}

func SmapToJson(data common.Smap) string {
	copyright := data["Copyright"]
	data["Copyright"] = "[skipped] (See 'Copyright' template for full text)"
	b, err := json.MarshalIndent(data, " ", "\t")
	data["Copyright"] = copyright
	common.ErrCheckExitf(err, 1, "error encoding data: %s", err)
	return fmt.Sprintf("%s", b)
}

func is_locked(sb_dir string) bool {
	return common.FileExists(sb_dir+"/no_clear") || common.FileExists(sb_dir+"/no_clear_all")
}

func CheckDirectory(sdef SandboxDef) SandboxDef {
	sandbox_dir := sdef.SandboxDir
	if common.DirExists(sandbox_dir) {
		if sdef.Force {
			if is_locked(sandbox_dir) {
				common.Exitf(1, "Sandbox in %s is locked. Cannot be overwritten\nYou can unlock it with 'dbdeployer admin unlock %s'\n", sandbox_dir, common.DirName(sandbox_dir))
			}
			fmt.Printf("Overwriting directory %s\n", sandbox_dir)
			stop_command := sandbox_dir + "/stop"
			if !common.ExecExists(stop_command) {
				stop_command = sandbox_dir + "/stop_all"
			}
			if !common.ExecExists(stop_command) {
				fmt.Printf("Neither 'stop' or 'stop_all' found in %s\n", sandbox_dir)
			}

			used_ports_list := common.GetInstalledPorts(sandbox_dir)
			my_used_ports := make(map[int]bool)
			for _, p := range used_ports_list {
				my_used_ports[p] = true
			}

			log_directory := getLogDirFromSbDescription(sandbox_dir)
			common.Run_cmd(stop_command)
			err, _ := common.Run_cmd_with_args("rm", []string{"-rf", sandbox_dir})
			common.ErrCheckExitf(err, 1, "Error while deleting sandbox %s", sandbox_dir)
			if log_directory != "" {
				err, _ = common.Run_cmd_with_args("rm", []string{"-rf", log_directory})
				common.ErrCheckExitf(err, 1, "Error while deleting log directory %s", log_directory)
			}
			var new_installed_ports []int
			for _, port := range sdef.InstalledPorts {
				if !my_used_ports[port] {
					new_installed_ports = append(new_installed_ports, port)
				}
			}
			sdef.InstalledPorts = new_installed_ports
		} else {
			common.Exitf(1, "Directory %s already exists. Use --force to override.", sandbox_dir)
		}
	}
	return sdef
}

func CheckPort(caller string, sandbox_type string, installed_ports []int, port int) {
	conflict := 0
	for _, p := range installed_ports {
		if p == port {
			conflict = p
		}
	}
	if conflict > 0 {
		common.Exitf(1, "Port conflict detected for %s (%s). Port %d is already used", sandbox_type, caller, conflict)
	}
}

func getmatch(key string, names []string, matches []string) string {
	if len(matches) < len(names) {
		return ""
	}
	for n, _ := range names {
		if names[n] == key {
			return matches[n]
		}
	}
	return ""
}

func FixServerUuid(sdef SandboxDef) (uuid_file, new_uuid string) {
	if !common.GreaterOrEqualVersion(sdef.Version, []int{5, 6, 9}) {
		return
	}
	new_uuid = fmt.Sprintf("server-uuid=%s", common.MakeCustomizedUuid(sdef.Port, sdef.NodeNum))
	operation_dir := sdef.SandboxDir + "/data"
	uuid_file = operation_dir + "/auto.cnf"
	return
}

func slice_to_text(s_array []string) string {
	var text string = ""
	for _, v := range s_array {
		if len(v) > 0 {
			text += fmt.Sprintf("%s\n", v)
		}
	}
	return text
}

func set_mysqlx_properties(sdef SandboxDef, global_tmp_dir string) SandboxDef {
	mysqlx_port := sdef.MysqlXPort
	if mysqlx_port == 0 {
		mysqlx_port = common.FindFreePort(sdef.Port+defaults.Defaults().MysqlXPortDelta, sdef.InstalledPorts, 1)
	}
	sdef.MyCnfOptions = append(sdef.MyCnfOptions, fmt.Sprintf("mysqlx-port=%d", mysqlx_port))
	sdef.MyCnfOptions = append(sdef.MyCnfOptions, fmt.Sprintf("mysqlx-socket=%s/mysqlx-%d.sock", global_tmp_dir, mysqlx_port))
	sdef.MorePorts = append(sdef.MorePorts, mysqlx_port)
	sdef.MysqlXPort = mysqlx_port
	return sdef
}

func CreateSingleSandbox(sdef SandboxDef) (exec_list []concurrent.ExecutionList) {

	var sandbox_dir string

	if sdef.SBType == "" {
		sdef.SBType = "single"
	}
	log_name := sdef.SBType
	if sdef.NodeNum > 0 {
		log_name = fmt.Sprintf("%s-%d", log_name, sdef.NodeNum)
	}
	fname, logger := defaults.NewLogger(common.LogDirName(), log_name)
	sdef.LogFileName = common.ReplaceLiteralHome(fname)
	logger.Printf("Single Sandbox Definition: %s\n", SandboxDefToJson(sdef))
	if !common.DirExists(sdef.Basedir) {
		common.Exitf(1, "Base directory %s does not exist", sdef.Basedir)
	}

	if sdef.Port <= 1024 {
		common.Exitf(1, "Port for sandbox must be > 1024 (given:%d)", sdef.Port)
	}

	version_fname := common.VersionToName(sdef.Version)
	if sdef.Prompt == "" {
		sdef.Prompt = "mysql"
	}
	if sdef.DirName == "" {
		if sdef.Version != sdef.BasedirName {
			sdef.DirName = defaults.Defaults().SandboxPrefix + sdef.BasedirName
		} else {
			sdef.DirName = defaults.Defaults().SandboxPrefix + version_fname
		}
	}
	sandbox_dir = sdef.SandboxDir + "/" + sdef.DirName
	sdef.SandboxDir = sandbox_dir
	logger.Printf("Single Sandbox directory defined as %s\n", sdef.SandboxDir)
	datadir := sandbox_dir + "/data"
	tmpdir := sandbox_dir + "/tmp"
	global_tmp_dir := os.Getenv("TMPDIR")
	if global_tmp_dir == "" {
		global_tmp_dir = "/tmp"
	}
	if !common.DirExists(global_tmp_dir) {
		common.Exitf(1, "TMP directory %s does not exist", global_tmp_dir)
	}
	if sdef.NodeNum == 0 && !sdef.Force {
		sdef.Port = common.FindFreePort(sdef.Port, sdef.InstalledPorts, 1)
		logger.Printf("Port defined as %d using FindFreePort \n", sdef.Port)
	}
	using_plugins := false
	right_plugin_dir := true // Assuming we can use the right plugin directory
	if sdef.EnableMysqlX {
		if !common.GreaterOrEqualVersion(sdef.Version, []int{5, 7, 12}) {
			common.Exit(1, "option --enable-mysqlx requires version 5.7.12+")
		}
		// If the version is 8.0.11 or later, MySQL X is enabled already
		if !common.GreaterOrEqualVersion(sdef.Version, []int{8, 0, 11}) {
			sdef.MyCnfOptions = append(sdef.MyCnfOptions, "plugin_load=mysqlx=mysqlx.so")
			sdef = set_mysqlx_properties(sdef, global_tmp_dir)
			logger.Printf("Added mysqlx plugin to my.cnf\n")
		}
		using_plugins = true
	}
	if common.GreaterOrEqualVersion(sdef.Version, []int{8, 0, 11}) && !sdef.DisableMysqlX {
		using_plugins = true
	}
	if sdef.ExposeDdTables {
		if !common.GreaterOrEqualVersion(sdef.Version, []int{8, 0, 0}) {
			common.Exit(1, "--expose-dd-tables requires MySQL 8.0.0+")
		}
		sdef.PostGrantsSql = append(sdef.PostGrantsSql, SingleTemplates["expose_dd_tables"].Contents)
		if sdef.CustomMysqld != "" && sdef.CustomMysqld != "mysqld-debug" {
			common.Exit(1,
				fmt.Sprintf("--expose-dd-tables requires mysqld-debug. A different file was indicated (--custom-mysqld=%s)", sdef.CustomMysqld),
				"Either use \"mysqld-debug\" or remove --custom-mysqld")
		}
		sdef.CustomMysqld = "mysqld-debug"
		logger.Printf("Using mysqld-debug for this sandbox\n")
	}
	if sdef.CustomMysqld != "" {
		custom_mysqld := sdef.Basedir + "/bin/" + sdef.CustomMysqld
		if !common.ExecExists(custom_mysqld) {
			common.Exit(1,
				fmt.Sprintf("File %s not found or not executable", custom_mysqld),
				fmt.Sprintf("The file \"%s\" (defined with --custom-mysqld) must be in the same directory as the regular mysqld", sdef.CustomMysqld))
		}
		plugin_debug_dir := fmt.Sprintf("%s/lib/plugin/debug", sdef.Basedir)
		if sdef.CustomMysqld == "mysqld-debug" && common.DirExists(plugin_debug_dir) {
			sdef.MyCnfOptions = append(sdef.MyCnfOptions, fmt.Sprintf("plugin-dir=%s", plugin_debug_dir))
		} else {
			right_plugin_dir = false
		}
	}
	if common.GreaterOrEqualVersion(sdef.Version, []int{5, 1, 0}) {
		if sdef.EnableGeneralLog {
			sdef.MyCnfOptions = append(sdef.MyCnfOptions, "general_log=1")
			logger.Printf("Enabling general log\n")
		}
		if sdef.InitGeneralLog {
			sdef.InitOptions = append(sdef.InitOptions, "--general_log=1")
			logger.Printf("Enabling general log during initialization\n")
		}
	}
	if common.GreaterOrEqualVersion(sdef.Version, []int{8, 0, 4}) {
		if sdef.NativeAuthPlugin == true {
			sdef.InitOptions = append(sdef.InitOptions, "--default_authentication_plugin=mysql_native_password")
			sdef.MyCnfOptions = append(sdef.MyCnfOptions, "default_authentication_plugin=mysql_native_password")
			logger.Printf("Using mysql_native_password for authentication\n")
		}
	}
	if common.GreaterOrEqualVersion(sdef.Version, []int{8, 0, 11}) {
		if sdef.DisableMysqlX {
			sdef.MyCnfOptions = append(sdef.MyCnfOptions, "mysqlx=OFF")
			logger.Printf("Disabling MySQLX\n")
		} else {
			sdef = set_mysqlx_properties(sdef, global_tmp_dir)
		}
	}
	mysqlsh_executable := fmt.Sprintf("%s/bin/mysqlsh", sdef.Basedir)
	if !common.ExecExists(mysqlsh_executable) {
		mysqlsh_executable = "mysqlsh"
	}
	if sdef.MyCnfFile != "" {
		options := GetOptionsFromFile(sdef.MyCnfFile)
		if len(options) > 0 {
			sdef.MyCnfOptions = append(sdef.MyCnfOptions, fmt.Sprintf("# options retrieved from %s", sdef.MyCnfFile))
		}
		for _, option := range options {
			// fmt.Printf("[%s]\n", option)
			sdef.MyCnfOptions = append(sdef.MyCnfOptions, option)
		}
	}
	if common.Includes(slice_to_text(sdef.MyCnfOptions), "plugin.load") {
		using_plugins = true
	}
	if common.Includes(sdef.SemiSyncOptions, "plugin.load") {
		using_plugins = true
	}
	if using_plugins {
		if !right_plugin_dir {
			common.Exit(1,
				"The request of using mysqld-debug can't be honored.",
				"This deployment is using a plugin, but the debug",
				"directory for plugins was not found")
		}
	}
	timestamp := time.Now()
	var data common.Smap = common.Smap{"Basedir": sdef.Basedir,
		"Copyright":            SingleTemplates["Copyright"].Contents,
		"AppVersion":           common.VersionDef,
		"DateTime":             timestamp.Format(time.UnixDate),
		"SandboxDir":           sandbox_dir,
		"CustomMysqld":         sdef.CustomMysqld,
		"Port":                 sdef.Port,
		"MysqlXPort":           sdef.MysqlXPort,
		"MysqlShell":           mysqlsh_executable,
		"BasePort":             sdef.BasePort,
		"Prompt":               sdef.Prompt,
		"Version":              sdef.Version,
		"Datadir":              datadir,
		"Tmpdir":               tmpdir,
		"GlobalTmpDir":         global_tmp_dir,
		"DbUser":               sdef.DbUser,
		"DbPassword":           sdef.DbPassword,
		"RplUser":              sdef.RplUser,
		"RplPassword":          sdef.RplPassword,
		"RemoteAccess":         sdef.RemoteAccess,
		"BindAddress":          sdef.BindAddress,
		"OsUser":               os.Getenv("USER"),
		"ReplOptions":          sdef.ReplOptions,
		"GtidOptions":          sdef.GtidOptions,
		"ReplCrashSafeOptions": sdef.ReplCrashSafeOptions,
		"SemiSyncOptions":      sdef.SemiSyncOptions,
		"ExtraOptions":         slice_to_text(sdef.MyCnfOptions),
		"ReportHost":           fmt.Sprintf("report-host=single-%d", sdef.Port),
		"ReportPort":           fmt.Sprintf("report-port=%d", sdef.Port),
		"HistoryDir":           sdef.HistoryDir,
	}
	if sdef.NodeNum != 0 {
		data["ReportHost"] = fmt.Sprintf("report-host = node-%d", sdef.NodeNum)
	}
	if sdef.SkipReportHost || sdef.SBType == "group-node" {
		data["ReportHost"] = ""
	}
	if sdef.SkipReportPort {
		data["ReportPort"] = ""
	}
	if sdef.ServerId > 0 {
		data["ServerId"] = fmt.Sprintf("server-id=%d", sdef.ServerId)
	} else {
		data["ServerId"] = ""
	}
	if common.DirExists(sandbox_dir) {
		sdef = CheckDirectory(sdef)
	}
	logger.Printf("Checking port %d using CheckPort\n", sdef.Port)
	CheckPort("CreateSingleSandbox", sdef.SBType, sdef.InstalledPorts, sdef.Port)

	//fmt.Printf("creating: %s\n", sandbox_dir)
	common.Mkdir(sandbox_dir)

	logger.Printf("Created directory %s\n", sdef.SandboxDir)
	logger.Printf("Single Sandbox template data: %s\n", SmapToJson(data))

	// fmt.Printf("creating: %s\n", datadir)
	common.Mkdir(datadir)
	logger.Printf("Created directory %s\n", datadir)
	// fmt.Printf("creating: %s\n", tmpdir)
	common.Mkdir(tmpdir)
	logger.Printf("Created directory %s\n", tmpdir)
	script := sdef.Basedir + "/scripts/mysql_install_db"
	init_script_flags := ""
	if common.GreaterOrEqualVersion(sdef.Version, []int{5, 7, 0}) {
		script = sdef.Basedir + "/bin/mysqld"
		init_script_flags = "--initialize-insecure"
	}
	// fmt.Printf("Script: %s\n", script)
	if !common.ExecExists(script) {
		common.Exitf(1, "Script '%s' not found", script)
	}
	if len(sdef.InitOptions) > 0 {
		for _, op := range sdef.InitOptions {
			init_script_flags += " " + op
		}
	}
	data["InitScript"] = script
	data["InitDefaults"] = "--no-defaults"
	if init_script_flags != "" {
		init_script_flags = fmt.Sprintf("\\\n    %s", init_script_flags)
	}
	data["ExtraInitFlags"] = init_script_flags
	data["FixUuidFile1"] = ""
	data["FixUuidFile2"] = ""

	if !sdef.KeepUuid {
		uuid_fname, new_uuid := FixServerUuid(sdef)
		if uuid_fname != "" {
			data["FixUuidFile1"] = fmt.Sprintf(`echo "[auto]" > %s`, uuid_fname)
			data["FixUuidFile2"] = fmt.Sprintf(`echo "%s" >> %s`, new_uuid, uuid_fname)
			logger.Printf("Created custom UUID %s\n", new_uuid)
		}
	}

	write_script(logger, SingleTemplates, "init_db", "init_db_template", sandbox_dir, data, true)
	if sdef.RunConcurrently {
		var eCommand = concurrent.ExecCommand{
			Cmd:  sandbox_dir + "/init_db",
			Args: []string{},
		}
		logger.Printf("Added init_db script to execution list\n")
		exec_list = append(exec_list, concurrent.ExecutionList{Logger: logger, Priority: 0, Command: eCommand})
	} else {
		logger.Printf("Running init_db script \n")
		err, _ := common.Run_cmd_ctrl(sandbox_dir+"/init_db", true)
		if err == nil {
			if !sdef.Multi {
				if defaults.UsingDbDeployer {
					fmt.Printf("Database installed in %s\n", common.ReplaceLiteralHome(sandbox_dir))
					fmt.Printf("run 'dbdeployer usage single' for basic instructions'\n")
				}
			}
		} else {
			fmt.Printf("err: %s\n", err)
		}
	}

	sb_item := defaults.SandboxItem{
		Origin:      sdef.Basedir,
		SBType:      sdef.SBType,
		Version:     sdef.Version,
		Port:        []int{sdef.Port},
		Nodes:       []string{},
		Destination: sandbox_dir,
	}

	if sdef.LogFileName != "" {
		sb_item.LogDirectory = common.DirName(sdef.LogFileName)
	}
	sb_desc := common.SandboxDescription{
		Basedir: sdef.Basedir,
		SBType:  sdef.SBType,
		Version: sdef.Version,
		Port:    []int{sdef.Port},
		Nodes:   0,
		NodeNum: sdef.NodeNum,
		LogFile: sdef.LogFileName,
	}
	if len(sdef.MorePorts) > 0 {
		for _, port := range sdef.MorePorts {
			sb_desc.Port = append(sb_desc.Port, port)
			sb_item.Port = append(sb_item.Port, port)
		}
	}
	logger.Printf("Writing single sandbox description\n")
	common.WriteSandboxDescription(sandbox_dir, sb_desc)
	if sdef.SBType == "single" {
		defaults.UpdateCatalog(sandbox_dir, sb_item)
	}
	logger.Printf("Writing single sandbox scripts\n")
	write_script(logger, SingleTemplates, "start", "start_template", sandbox_dir, data, true)
	write_script(logger, SingleTemplates, "status", "status_template", sandbox_dir, data, true)
	write_script(logger, SingleTemplates, "stop", "stop_template", sandbox_dir, data, true)
	write_script(logger, SingleTemplates, "clear", "clear_template", sandbox_dir, data, true)
	write_script(logger, SingleTemplates, "use", "use_template", sandbox_dir, data, true)
	if sdef.MysqlXPort != 0 {
		write_script(logger, SingleTemplates, "mysqlsh", "mysqlsh_template", sandbox_dir, data, true)
	}
	write_script(logger, SingleTemplates, "show_log", "show_log_template", sandbox_dir, data, true)
	write_script(logger, SingleTemplates, "send_kill", "send_kill_template", sandbox_dir, data, true)
	write_script(logger, SingleTemplates, "restart", "restart_template", sandbox_dir, data, true)
	write_script(logger, SingleTemplates, "load_grants", "load_grants_template", sandbox_dir, data, true)
	write_script(logger, SingleTemplates, "add_option", "add_option_template", sandbox_dir, data, true)
	write_script(logger, SingleTemplates, "my", "my_template", sandbox_dir, data, true)
	write_script(logger, SingleTemplates, "show_binlog", "show_binlog_template", sandbox_dir, data, true)
	write_script(logger, SingleTemplates, "show_relaylog", "show_relaylog_template", sandbox_dir, data, true)
	write_script(logger, SingleTemplates, "test_sb", "test_sb_template", sandbox_dir, data, true)

	write_script(logger, SingleTemplates, "my.sandbox.cnf", "my_cnf_template", sandbox_dir, data, false)
	switch {
	case common.GreaterOrEqualVersion(sdef.Version, []int{8, 0, 0}):
		write_script(logger, SingleTemplates, "grants.mysql", "grants_template8x", sandbox_dir, data, false)
	case common.GreaterOrEqualVersion(sdef.Version, []int{5, 7, 6}):
		write_script(logger, SingleTemplates, "grants.mysql", "grants_template57", sandbox_dir, data, false)
	default:
		write_script(logger, SingleTemplates, "grants.mysql", "grants_template5x", sandbox_dir, data, false)
	}
	write_script(logger, SingleTemplates, "sb_include", "sb_include_template", sandbox_dir, data, false)

	pre_grant_sql_file := sandbox_dir + "/pre_grants.sql"
	post_grant_sql_file := sandbox_dir + "/post_grants.sql"
	if sdef.PreGrantsSqlFile != "" {
		common.CopyFile(sdef.PreGrantsSqlFile, pre_grant_sql_file)
	}
	if sdef.PostGrantsSqlFile != "" {
		common.CopyFile(sdef.PostGrantsSqlFile, post_grant_sql_file)
	}

	if len(sdef.PreGrantsSql) > 0 {
		if common.FileExists(pre_grant_sql_file) {
			common.AppendStrings(sdef.PreGrantsSql, pre_grant_sql_file, ";")
		} else {
			common.WriteStrings(sdef.PreGrantsSql, pre_grant_sql_file, ";")
		}
	}
	if len(sdef.PostGrantsSql) > 0 {
		if common.FileExists(post_grant_sql_file) {
			common.AppendStrings(sdef.PostGrantsSql, post_grant_sql_file, ";")
		} else {
			common.WriteStrings(sdef.PostGrantsSql, post_grant_sql_file, ";")
		}
	}
	//common.Run_cmd(sandbox_dir + "/start", []string{})
	if !sdef.SkipStart && sdef.RunConcurrently {
		var eCommand2 = concurrent.ExecCommand{
			Cmd:  sandbox_dir + "/start",
			Args: []string{},
		}
		logger.Printf("Adding start command to execution list\n")
		exec_list = append(exec_list, concurrent.ExecutionList{Logger: logger, Priority: 2, Command: eCommand2})
		if sdef.LoadGrants {
			var eCommand3 = concurrent.ExecCommand{
				Cmd:  sandbox_dir + "/load_grants",
				Args: []string{"pre_grants.sql"},
			}
			var eCommand4 = concurrent.ExecCommand{
				Cmd:  sandbox_dir + "/load_grants",
				Args: []string{},
			}
			var eCommand5 = concurrent.ExecCommand{
				Cmd:  sandbox_dir + "/load_grants",
				Args: []string{"post_grants.sql"},
			}
			logger.Printf("Adding pre grants command to execution list\n")
			logger.Printf("Adding load grants command to execution list\n")
			logger.Printf("Adding post grants command to execution list\n")
			exec_list = append(exec_list, concurrent.ExecutionList{Logger: logger, Priority: 3, Command: eCommand3})
			exec_list = append(exec_list, concurrent.ExecutionList{Logger: logger, Priority: 4, Command: eCommand4})
			exec_list = append(exec_list, concurrent.ExecutionList{Logger: logger, Priority: 5, Command: eCommand5})
		}
	} else {
		if !sdef.SkipStart {
			logger.Printf("Running start script\n")
			common.Run_cmd(sandbox_dir + "/start")
			if sdef.LoadGrants {
				logger.Printf("Running pre grants script\n")
				common.Run_cmd_with_args(sandbox_dir+"/load_grants", []string{"pre_grants.sql"})
				logger.Printf("Running load grants script\n")
				common.Run_cmd(sandbox_dir + "/load_grants")
				logger.Printf("Running post grants script\n")
				common.Run_cmd_with_args(sandbox_dir+"/load_grants", []string{"post_grants.sql"})
			}
		}
	}
	return
}

func write_script(logger *defaults.Logger, temp_var TemplateCollection, name, template_name, directory string, data common.Smap, make_executable bool) {
	template := temp_var[template_name].Contents
	template = common.TrimmedLines(template)
	data["TemplateName"] = template_name
	text := common.Tprintf(template, data)
	executable_status := ""
	if make_executable {
		write_exec(name, text, directory)
		executable_status = " executable"
	} else {
		write_regular_file(name, text, directory)
	}
	if logger != nil {
		logger.Printf("Creating%s script '%s/%s' using template '%s'\n", executable_status, common.ReplaceLiteralHome(directory), name, template_name)
	}
}

func write_exec(filename, text, directory string) {
	fname := write_regular_file(filename, text, directory)
	os.Chmod(fname, 0744)
}

func write_regular_file(filename, text, directory string) string {
	fname := directory + "/" + filename
	common.WriteString(text, fname)
	return fname
}

func getLogDirFromSbDescription(full_path string) string {
	sbdescription := full_path + "/sbdescription.json"
	log_file := ""
	log_directory := ""
	if common.FileExists(sbdescription) {
		sbd := common.ReadSandboxDescription(full_path)
		log_file = sbd.LogFile
		if log_file != "" {
			log_file = common.ReplaceHomeVar(log_file)
			log_directory = common.DirName(log_file)
			if !common.DirExists(log_directory) {
				log_directory = ""
			}
		}
	}
	return log_directory
}

func RemoveSandbox(sandbox_dir, sandbox string, run_concurrently bool) (exec_list []concurrent.ExecutionList) {
	full_path := sandbox_dir + "/" + sandbox
	if !common.DirExists(full_path) {
		common.Exitf(1, "Directory '%s' not found", full_path)
	}
	preserve := full_path + "/no_clear_all"
	if !common.ExecExists(preserve) {
		preserve = full_path + "/no_clear"
	}
	if common.ExecExists(preserve) {
		fmt.Printf("The sandbox %s is locked\n", sandbox)
		fmt.Printf("You need to unlock it with \"dbdeployer admin unlock\"\n")
		return
	}
	log_directory := getLogDirFromSbDescription(full_path)
	stop := full_path + "/stop_all"
	if !common.ExecExists(stop) {
		stop = full_path + "/stop"
	}
	if !common.ExecExists(stop) {
		common.Exitf(1, "Executable '%s' not found", stop)
	}

	if run_concurrently {
		var eCommand1 = concurrent.ExecCommand{
			Cmd:  stop,
			Args: []string{},
		}
		exec_list = append(exec_list, concurrent.ExecutionList{Logger: nil, Priority: 0, Command: eCommand1})
	} else {
		if defaults.UsingDbDeployer {
			fmt.Printf("Running %s\n", stop)
		}
		err, _ := common.Run_cmd(stop)
		common.ErrCheckExitf(err, 1, "Error while stopping sandbox %s", full_path)
	}

	rm_targets := []string{full_path, log_directory}

	for _, target := range rm_targets {
		if target == "" {
			continue
		}
		cmd_str := "rm"
		rm_args := []string{"-rf", target}
		if run_concurrently {
			var eCommand2 = concurrent.ExecCommand{
				Cmd:  cmd_str,
				Args: rm_args,
			}
			exec_list = append(exec_list, concurrent.ExecutionList{Logger: nil, Priority: 1, Command: eCommand2})
		} else {
			for _, item := range rm_args {
				cmd_str += " " + item
			}
			if defaults.UsingDbDeployer && target != log_directory {
				fmt.Printf("Running %s\n", cmd_str)
			}
			err, _ := common.Run_cmd_with_args("rm", rm_args)
			common.ErrCheckExitf(err, 1, "Error while deleting directory %s", target)
			if defaults.UsingDbDeployer && target != log_directory {
				fmt.Printf("Directory %s deleted\n", target)
			}
		}
	}
	// fmt.Printf("%#v\n",exec_list)
	return
}
