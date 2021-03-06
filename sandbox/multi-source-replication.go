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
	"fmt"
	"github.com/datacharmer/dbdeployer/common"
	"github.com/datacharmer/dbdeployer/defaults"
	"regexp"
	"strconv"
	"strings"
)

func check_node_lists(nodes int, mlist, slist []int) {
	for _, N := range mlist {
		if N > nodes {
			common.Exitf(1, "Master num '%d' greater than number of nodes (%d)", N, nodes)
		}
	}
	for _, N := range slist {
		if N > nodes {
			common.Exitf(1, "Slave num '%d' greater than number of nodes (%d)", N, nodes)
		}
	}
	for _, M := range mlist {
		for _, S := range slist {
			if S == M {
				common.Exitf(1, "Overlapping values: %d is in both master and slave list", M)
			}
		}
	}
	total_nodes := len(mlist) + len(slist)
	if total_nodes != nodes {
		common.Exitf(1, "Mismatched values: masters (%d) + slaves (%d) = %d. Expected: %d", len(mlist), len(slist), total_nodes, nodes)
	}
}

func nodes_list_to_int_slice(nodes_list string, nodes int) (int_list []int) {
	separator := " "
	if common.Includes(nodes_list, ",") {
		separator = ","
	} else if common.Includes(nodes_list, ":") {
		separator = ":"
	} else if common.Includes(nodes_list, ";") {
		separator = ";"
	} else if common.Includes(nodes_list, `\.`) {
		separator = "."
	} else {
		separator = " "
	}
	list := strings.Split(nodes_list, separator)
	// fmt.Printf("# separator: <%s> %#v\n",separator, list)
	if len(list) == 0 {
		common.Exitf(1, "Empty nodes list given (%s)", nodes_list)
	}
	for _, s := range list {
		if s != "" {
			num, err := strconv.Atoi(s)
			common.ErrCheckExitf(err, 1, "Error converting node number '%s' to int", s)
			int_list = append(int_list, num)
		}
	}
	if len(int_list) == 0 {
		fmt.Printf("List '%s' is empty\n", nodes_list)
	}
	if len(int_list) > nodes {
		fmt.Printf("List '%s' is greater than the expected number of nodes (%d)\n", nodes_list, nodes)
	}
	return
}

func make_nodes_list(nodes int) (nodes_list string) {
	for N := 1; N <= nodes; N++ {
		nodes_list += fmt.Sprintf("%d ", N)
	}
	return nodes_list
}

func CreateAllMastersReplication(sdef SandboxDef, origin string, nodes int, master_ip string) {
	sdef.SBType = "all-masters"

	fname, logger := defaults.NewLogger(common.LogDirName(), "all-masters")
	sdef.LogFileName = common.ReplaceLiteralHome(fname)
	sdef.Logger = logger
	sdef.GtidOptions = SingleTemplates["gtid_options_57"].Contents
	sdef.ReplCrashSafeOptions = SingleTemplates["repl_crash_safe_options"].Contents
	if sdef.DirName == "" {
		sdef.DirName += defaults.Defaults().AllMastersPrefix + common.VersionToName(origin)
	}
	sandbox_dir := sdef.SandboxDir
	sdef.SandboxDir = common.DirName(sdef.SandboxDir)
	if sdef.BasePort == 0 {
		sdef.BasePort = defaults.Defaults().AllMastersReplicationBasePort
	}
	master_abbr := defaults.Defaults().MasterAbbr
	slave_abbr := defaults.Defaults().SlaveAbbr
	master_label := defaults.Defaults().MasterName
	slave_label := defaults.Defaults().SlavePrefix
	data := CreateMultipleSandbox(sdef, origin, nodes)

	sdef.SandboxDir = data["SandboxDir"].(string)
	master_list := make_nodes_list(nodes)
	slist := nodes_list_to_int_slice(master_list, nodes)
	data["MasterIp"] = master_ip
	data["MasterAbbr"] = master_abbr
	data["MasterLabel"] = master_label
	data["MasterList"] = normalize_node_list(master_list)
	data["SlaveAbbr"] = slave_abbr
	data["SlaveLabel"] = slave_label
	data["SlaveList"] = normalize_node_list(master_list)
	data["RplUser"] = sdef.RplUser
	data["RplPassword"] = sdef.RplPassword
	data["NodeLabel"] = defaults.Defaults().NodePrefix
	logger.Printf("Writing master and slave scripts in %s\n", sdef.SandboxDir)
	for _, node := range slist {
		data["Node"] = node
		write_script(logger, ReplicationTemplates, fmt.Sprintf("s%d", node), "slave_template", sandbox_dir, data, true)
		write_script(logger, ReplicationTemplates, fmt.Sprintf("m%d", node), "slave_template", sandbox_dir, data, true)
	}
	logger.Printf("Writing all-masters replication scripts in %s\n", sdef.SandboxDir)
	write_script(logger, ReplicationTemplates, "test_replication", "multi_source_test_template", sandbox_dir, data, true)
	write_script(logger, ReplicationTemplates, "use_all_slaves", "multi_source_use_slaves_template", sandbox_dir, data, true)
	write_script(logger, ReplicationTemplates, "use_all_masters", "multi_source_use_masters_template", sandbox_dir, data, true)
	write_script(logger, ReplicationTemplates, "check_ms_nodes", "check_multi_source_template", sandbox_dir, data, true)
	write_script(logger, ReplicationTemplates, "initialize_ms_nodes", "multi_source_template", sandbox_dir, data, true)
	if !sdef.SkipStart {
		logger.Printf("Initializing all-masters replication \n")
		fmt.Println(common.ReplaceLiteralHome(sandbox_dir) + "/initialize_ms_nodes")
		common.Run_cmd(sandbox_dir + "/initialize_ms_nodes")
	}
}

func normalize_node_list(list string) string {
	re := regexp.MustCompile(`[,:\.]`)
	return re.ReplaceAllString(list, " ")
}

func CreateFanInReplication(sdef SandboxDef, origin string, nodes int, master_ip, master_list, slave_list string) {
	sdef.SBType = "fan-in"

	fname, logger := defaults.NewLogger(common.LogDirName(), "fan-in")
	sdef.LogFileName = fname
	sdef.Logger = logger
	sdef.GtidOptions = SingleTemplates["gtid_options_57"].Contents
	sdef.ReplCrashSafeOptions = SingleTemplates["repl_crash_safe_options"].Contents
	if sdef.DirName == "" {
		sdef.DirName = defaults.Defaults().FanInPrefix + common.VersionToName(origin)
	}
	if sdef.BasePort == 0 {
		sdef.BasePort = defaults.Defaults().FanInReplicationBasePort
	}
	sandbox_dir := sdef.SandboxDir
	sdef.SandboxDir = common.DirName(sdef.SandboxDir)
	mlist := nodes_list_to_int_slice(master_list, nodes)
	slist := nodes_list_to_int_slice(slave_list, nodes)
	check_node_lists(nodes, mlist, slist)
	data := CreateMultipleSandbox(sdef, origin, nodes)

	sdef.SandboxDir = data["SandboxDir"].(string)
	master_abbr := defaults.Defaults().MasterAbbr
	slave_abbr := defaults.Defaults().SlaveAbbr
	master_label := defaults.Defaults().MasterName
	slave_label := defaults.Defaults().SlavePrefix
	data["MasterList"] = normalize_node_list(master_list)
	data["SlaveList"] = normalize_node_list(slave_list)
	data["MasterAbbr"] = master_abbr
	data["MasterLabel"] = master_label
	data["SlaveAbbr"] = slave_abbr
	data["SlaveLabel"] = slave_label
	data["RplUser"] = sdef.RplUser
	data["RplPassword"] = sdef.RplPassword
	data["NodeLabel"] = defaults.Defaults().NodePrefix
	data["MasterIp"] = master_ip
	logger.Printf("Writing master and slave scripts in %s\n", sdef.SandboxDir)
	for _, slave := range slist {
		data["Node"] = slave
		write_script(logger, ReplicationTemplates, fmt.Sprintf("s%d", slave), "slave_template", sandbox_dir, data, true)
	}
	for _, master := range mlist {
		data["Node"] = master
		write_script(logger, ReplicationTemplates, fmt.Sprintf("m%d", master), "slave_template", sandbox_dir, data, true)
	}
	logger.Printf("writing fan-in replication scripts in %s\n", sdef.SandboxDir)
	write_script(logger, ReplicationTemplates, "test_replication", "multi_source_test_template", sandbox_dir, data, true)
	write_script(logger, ReplicationTemplates, "check_ms_nodes", "check_multi_source_template", sandbox_dir, data, true)
	write_script(logger, ReplicationTemplates, "use_all_slaves", "multi_source_use_slaves_template", sandbox_dir, data, true)
	write_script(logger, ReplicationTemplates, "use_all_masters", "multi_source_use_masters_template", sandbox_dir, data, true)
	write_script(logger, ReplicationTemplates, "initialize_ms_nodes", "multi_source_template", sandbox_dir, data, true)
	if !sdef.SkipStart {
		logger.Printf("Initializing fan-in replication\n")
		fmt.Println(common.ReplaceLiteralHome(sandbox_dir) + "/initialize_ms_nodes")
		common.Run_cmd(sandbox_dir + "/initialize_ms_nodes")
	}
}
