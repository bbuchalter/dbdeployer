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

// Templates for replication

var (
	init_slaves_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}

# Don't use directly.
# This script is called by 'start_all' when needed
SBDIR={{.SandboxDir}}
cd $SBDIR
# workaround for Bug#89959
$SBDIR/{{.MasterLabel}}/use -h {{.MasterIp}} -u {{.RplUser}} -p{{.RplPassword}} -e 'set @a=1'
if [ ! -f needs_initialization ]
then
	# First run: root is running without password
	export NOPASSWORD=1
fi

{{ range .Slaves }}
echo "initializing {{.SlaveLabel}} {{.Node}}"
echo 'CHANGE MASTER TO  master_host="{{.MasterIp}}",  master_port={{.MasterPort}},  master_user="{{.RplUser}}",  master_password="{{.RplPassword}}" {{.ChangeMasterExtra}}' | $SBDIR/{{.NodeLabel}}{{.Node}}/use -u root
$SBDIR/{{.NodeLabel}}{{.Node}}/use -u root -e 'START SLAVE'
{{end}}
if [ -x ./post_initialization ]
then
    unset NOPASSWORD
    ./post_initialization > post_initialization.log 2>&1
	exit_code=$?
	if [ "$exit_code" == "0" ]
	then
		rm -f ./post_initialization
	fi
fi
`
	semi_sync_start_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}

# This script is called by 'initialize_slaves' when needed
SBDIR={{.SandboxDir}}
sleep 2
set -x
cd $SBDIR
$SBDIR/{{.MasterLabel}}/use -u root -e 'set global rpl_semi_sync_master_enabled=1'
echo "rpl_semi_sync_master_enabled=1" >> $SBDIR/{{.MasterLabel}}/my.sandbox.cnf
{{ range .Slaves }}
$SBDIR/{{.NodeLabel}}{{.Node}}/use -u root -e 'STOP SLAVE'
$SBDIR/{{.NodeLabel}}{{.Node}}/use -u root -e 'set global rpl_semi_sync_slave_enabled=1'
$SBDIR/{{.NodeLabel}}{{.Node}}/use -u root -e 'START SLAVE'
echo "rpl_semi_sync_slave_enabled=1" >> $SBDIR/{{.NodeLabel}}{{.Node}}/my.sandbox.cnf
{{end}}
`

	start_all_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
echo "# executing 'start' on $SBDIR"
echo "executing 'start' on {{.MasterLabel}}"
$SBDIR/{{.MasterLabel}}/start "$@"
{{ range .Slaves }}
echo "executing 'start' on {{.SlaveLabel}} {{.Node}}"
$SBDIR/{{.NodeLabel}}{{.Node}}/start "$@"
{{end}}
if [ -f $SBDIR/needs_initialization ]
then
	$SBDIR/initialize_{{.SlaveLabel}}s
    rm -f $SBDIR/needs_initialization
fi
`
	restart_all_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
$SBDIR/stop_all
$SBDIR/start_all "$@"
`
	use_all_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
if [ "$1" = "" ]
then
  echo "syntax: $0 command"
  exit 1
fi

if [ -z "$ONLY_SLAVES" -o -n "$ONLY_MASTER" ] 
then
	echo "# {{.MasterLabel}}  "
	echo "$@" | $SBDIR/{{.MasterLabel}}/use $MYCLIENT_OPTIONS
fi

if [ -z "$ONLY_MASTER" -o -n "$ONLY_SLAVES" ]
then
{{range .Slaves}}
	echo "# server: {{.Node}} "
	echo "$@" | $SBDIR/{{.NodeLabel}}{{.Node}}/use $MYCLIENT_OPTIONS
{{end}}
fi
`
	use_all_slaves_template string = `#!/bin/bash
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
unset ONLY_MASTER
export ONLY_SLAVES=1
$SBDIR/use_all "$@"
`
	use_all_masters_template string = `#!/bin/bash
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
unset ONLY_SLAVES
export ONLY_MASTER=1
$SBDIR/use_all "$@"
`
	stop_all_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
echo "# executing 'stop' on $SBDIR"
{{ range .Slaves }}
# echo 'executing "stop" on {{.SlaveLabel}} {{.Node}}'
$SBDIR/{{.NodeLabel}}{{.Node}}/stop "$@"
{{end}}
# echo 'executing "stop" on {{.MasterLabel}}'
$SBDIR/{{.MasterLabel}}/stop "$@"
`
	send_kill_all_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
echo "# executing 'send_kill' on $SBDIR"
{{ range .Slaves }}
echo 'executing "send_kill" on {{.SlaveLabel}} {{.Node}}'
$SBDIR/{{.NodeLabel}}{{.Node}}/send_kill "$@"
{{end}}
echo 'executing "send_kill" on {{.MasterLabel}}'
$SBDIR/{{.MasterLabel}}/send_kill "$@"
`
	clear_all_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
echo "# executing 'clear' on $SBDIR"
{{range .Slaves}}
echo 'executing "clear" on {{.SlaveLabel}} {{.Node}}'
$SBDIR/{{.NodeLabel}}{{.Node}}/clear "$@"
{{end}}
echo 'executing "clear" on {{.MasterLabel}}'
$SBDIR/{{.MasterLabel}}/clear "$@"
date > $SBDIR/needs_initialization
`
	status_all_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
echo "REPLICATION  $SBDIR"
mstatus=$($SBDIR/{{.MasterLabel}}/status)
if [ -f $SBDIR/{{.MasterLabel}}/data/mysql_sandbox{{.MasterPort}}.pid ]
then
	mport=$($SBDIR/{{.MasterLabel}}/use -BN -e "show variables like 'port'")
fi
echo "{{.MasterLabel}} : $mstatus  -  $mport ({{.MasterPort}})"
{{ range .Slaves }}
nstatus=$($SBDIR/{{.NodeLabel}}{{.Node}}/status )
if [ -f $SBDIR/{{.NodeLabel}}{{.Node}}/data/mysql_sandbox{{.NodePort}}.pid ]
then
	nport=$($SBDIR/{{.NodeLabel}}{{.Node}}/use -BN -e "show variables like 'port'")
fi
echo "{{.NodeLabel}}{{.Node}} : $nstatus  -  $nport ({{.NodePort}})"
{{end}}
`
	test_sb_all_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
echo "# executing 'test_sb' on $SBDIR"
echo 'executing "test_sb" on {{.MasterLabel}}'
$SBDIR/{{.MasterLabel}}/test_sb "$@"
exit_code=$?
if [ "$exit_code" != "0" ] ; then exit $exit_code ; fi
{{ range .Slaves }}
echo 'executing "test_sb" on {{.SlaveLabel}} {{.Node}}'
$SBDIR/{{.NodeLabel}}{{.Node}}/test_sb "$@"
exit_code=$?
if [ "$exit_code" != "0" ] ; then exit $exit_code ; fi
{{end}}
`

	check_slaves_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
echo "{{.MasterLabel}}"
port=$($SBDIR/{{.MasterLabel}}/use -BN -e "show variables like 'port'")
server_id=$($SBDIR/{{.MasterLabel}}/use -BN -e "show variables like 'server_id'")
echo "$port - $server_id"
$SBDIR/{{.MasterLabel}}/use -e 'show master status\G' | grep "File\|Position\|Executed"
{{ range .Slaves }}
echo "{{.SlaveLabel}}{{.Node}}"
port=$($SBDIR/{{.NodeLabel}}{{.Node}}/use -BN -e "show variables like 'port'")
server_id=$($SBDIR/{{.NodeLabel}}{{.Node}}/use -BN -e "show variables like 'server_id'")
echo "$port - $server_id"
$SBDIR/{{.NodeLabel}}{{.Node}}/use -e 'show slave status\G' | grep "\(Running:\|Master_Log_Pos\|\<Master_Log_File\|Retrieved\|Executed\)"
{{end}}
`
	master_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}

{{.SandboxDir}}/{{.MasterLabel}}/use "$@"
`
	slave_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}

{{.SandboxDir}}/{{.NodeLabel}}{{.Node}}/use "$@"
`
	test_replication_template string = `#!/bin/bash
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
cd $SBDIR

if [ -x ./{{.MasterAbbr}} ]
then
    MASTER=./{{.MasterAbbr}}
elif [ -x ./n1 ]
then
    MASTER=./n1
else
    echo "# No {{.MasterLabel}} found"
    exit 1
fi
$MASTER -e 'create schema if not exists test'
$MASTER test -e 'drop table if exists t1'
$MASTER test -e 'create table t1 (i int not null primary key, msg varchar(50), d date, t time, dt datetime, ts timestamp)'
#$MASTER test -e "insert into t1 values (1, 'test sandbox 1', '2015-07-16', '11:23:40','2015-07-17 12:34:50', null)"
#$MASTER test -e "insert into t1 values (2, 'test sandbox 2', '2015-07-17', '11:23:41','2015-07-17 12:34:51', null)"
for N in $(seq -f '%02.0f' 1 20)
do
    #echo "$MASTER test -e \"insert into t1 values ($N, 'test sandbox $N', '2015-07-$N', '11:23:$N','2015-07-17 12:34:$N', null)\""
    $MASTER test -e "insert into t1 values ($N, 'test sandbox $N', '2015-07-$N', '11:23:$N','2015-07-17 12:34:$N', null)"
done
sleep 0.5
MASTER_RECS=$($MASTER -BN -e 'select count(*) from test.t1')

master_status=master_status$$
slave_status=slave_status$$
$MASTER -e 'show master status\G' > $master_status
master_binlog=$(grep 'File:' $master_status | awk '{print $2}' )
master_pos=$(grep 'Position:' $master_status | awk '{print $2}' )
echo "# {{.MasterLabel}} log: $master_binlog - Position: $master_pos - Rows: $MASTER_RECS"
rm -f $master_status

FAILED=0
PASSED=0

function ok_equal
{
    fact="$1"
    expected="$2"
    msg="$3"
    if [ "$fact" == "$expected" ]
    then
        echo -n "ok"
        PASSED=$(($PASSED+1))
    else
        echo -n "not ok - (expected: <$expected> found: <$fact>) "
        FAILED=$(($FAILED+1))
    fi
    echo " - $msg"
}

function test_summary
{
    TESTS=$(($PASSED+$FAILED))
    if [ -n "$TAP_TEST" ]
    then
        echo "1..$TESTS"
    else
        PERCENT_PASSED=$(($PASSED/$TESTS*100))
        PERCENT_FAILED=$(($FAILED/$TESTS*100))
        printf "# Tests : %5d\n" $TESTS
    fi
    exit_code=0
	fail_label="failed"
	pass_label="PASSED"
    if [ "$FAILED" != "0" ]
    then
        fail_label="FAILED"
		pass_label="passed"
        exit_code=1
    fi
    printf "# $fail_label: %5d (%5.1f%%)\n" $FAILED $PERCENT_FAILED
    printf "# $pass_label: %5d (%5.1f%%)\n" $PASSED $PERCENT_PASSED
    echo "# exit code: $exit_code"
    exit $exit_code
}

for SLAVE_N in 1 2 3 4 5 6 7 8 9
do
    N=$(($SLAVE_N+1))
    unset SLAVE
    if [ -x ./{{.SlaveAbbr}}$SLAVE_N ]
    then
        SLAVE=./{{.SlaveAbbr}}$SLAVE_N
    elif [ -x ./n$N ]
    then
        SLAVE=./n$N
    fi
    if [ -n "$SLAVE" ]
    then
        echo "# Testing {{.SlaveLabel}} #$SLAVE_N"
        if [ -f initialize_nodes ]
        then
            sleep 3
        else
            S_READY=$($SLAVE -BN -e "select master_pos_wait('$master_binlog', $master_pos,60)")
            # master_pos_wait can return 0 or a positive number for successful replication
            # Any result that is not NULL or -1 is acceptable
            if [ "$S_READY" != "-1" -a "$S_READY" != "NULL" ]
            then
                S_READY=0
            fi
            ok_equal $S_READY 0 "{{.SlaveLabel}} #$SLAVE_N acknowledged reception of transactions from {{.MasterLabel}}"
        fi
		if [ -f initialize_{{.SlaveLabel}}s ]
		then
			$SLAVE -e 'show slave status\G' > $slave_status
			IO_RUNNING=$(grep -w Slave_IO_Running $slave_status | awk '{print $2}')
			ok_equal $IO_RUNNING Yes "{{.SlaveLabel}} #$SLAVE_N IO thread is running"
			SQL_RUNNING=$(grep -w Slave_IO_Running $slave_status | awk '{print $2}')
			ok_equal $SQL_RUNNING Yes "{{.SlaveLabel}} #$SLAVE_N SQL thread is running"
			rm -f $slave_status
		fi
        [ $FAILED == 0 ] || exit 1

        T1_EXISTS=$($SLAVE -BN -e 'show tables from test like "t1"')
        ok_equal $T1_EXISTS t1 "Table t1 found on {{.SlaveLabel}} #$SLAVE_N"
        T1_RECS=$($SLAVE -BN -e 'select count(*) from test.t1')
        ok_equal $T1_RECS $MASTER_RECS "Table t1 has $MASTER_RECS rows on #$SLAVE_N"
    fi
done
test_summary

`
	multi_source_template string = `#!/bin/bash
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
cd $SBDIR

$SBDIR/use_all 'reset master'

MASTERS="{{.MasterList}}"
SLAVES="{{.SlaveList}}"

for N in $SLAVES
do
    user_cmd=''
    for master in $MASTERS
    do
        if [ "$master" != "$N" ]
        then
            master_port=$($SBDIR/n$master -BN -e 'select @@port')
            $SBDIR/n$master -BN  -h {{.MasterIp}} --port=$master_port -u {{.RplUser}} -p{{.RplPassword}} -e 'set @a=1'
            user_cmd="$user_cmd CHANGE MASTER TO MASTER_USER='{{.RplUser}}', "
            user_cmd="$user_cmd MASTER_PASSWORD='{{.RplPassword}}', master_host='{{.MasterIp}}', "
            user_cmd="$user_cmd master_port=$master_port FOR CHANNEL '{{.NodeLabel}}$master';"
            user_cmd="$user_cmd START SLAVE FOR CHANNEL '{{.NodeLabel}}$master';"
        fi
    done
	VERBOSE_SQL=""
	if [ -n "$VERBOSE_SQL" ]
	then
		VERBOSE_SQL="-v"
	fi
    $SBDIR/{{.NodeLabel}}$N/use $VERBOSE_SQL -u root -e "$user_cmd"
done
`
	multi_source_use_slaves_template string = `#!/bin/bash
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
cd $SBDIR

MASTERS="{{.MasterList}}"
SLAVES="{{.SlaveList}}"

for N in $SLAVES
do
	echo "# server: $N"
	echo "$@" | $SBDIR/{{.NodeLabel}}$N/use $MYCLIENT_OPTIONS
done
`
	multi_source_use_masters_template string = `#!/bin/bash
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
cd $SBDIR

MASTERS="{{.MasterList}}"

for N in $MASTERS
do
	echo "# server: $N"
	echo "$@" | $SBDIR/{{.NodeLabel}}$N/use $MYCLIENT_OPTIONS
done
`

	check_multi_source_template string = `#!/bin/sh
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}

MASTERS="{{.MasterList}}"
SLAVES="{{.SlaveList}}"

for M in $MASTERS
do
	echo "# Master $M"
	port=$($SBDIR/{{.NodeLabel}}$M/use -BN -e "show variables like 'port'")
	server_id=$($SBDIR/{{.NodeLabel}}$M/use -BN -e "show variables like 'server_id'")
	echo "$port - $server_id"
	$SBDIR/{{.NodeLabel}}$M/use -e 'show master status\G' | grep "File\|Position\|Executed"
done
for S in $SLAVES
do
	echo "# Slave $S"
	port=$($SBDIR/{{.NodeLabel}}$S/use -BN -e "show variables like 'port'")
	server_id=$($SBDIR/{{.NodeLabel}}$S/use -BN -e "show variables like 'server_id'")
	echo "$port - $server_id"
	$SBDIR/{{.NodeLabel}}$S/use -e 'show slave status\G' | grep "\(Running:\|Master_Log_Pos\|\<Master_Log_File\|Retrieved\|Channel\|Executed\)"
done
`
	multi_source_test_template string = `#!/bin/bash
{{.Copyright}}
# Generated by dbdeployer {{.AppVersion}} using {{.TemplateName}} on {{.DateTime}}
SBDIR={{.SandboxDir}}
cd $SBDIR

pass=0
fail=0

function ok_equal {
	value=$1
	expected=$2
	message=$3
	if [ "$value" == "$expected" ]
	then
		echo "ok - '$value' == '$expected' - $message"
		pass=$((pass+1))
	else
		echo "NOT OK - found: '$value' expected: '$expected' - $message"
		fail=$((fail+1))
	fi
}

MASTERS="{{.MasterList}}"
SLAVES="{{.SlaveList}}"
total_tables=0
[ -z "$SLEEP_TIME" ] && SLEEP_TIME=1

for M in $MASTERS
do
	echo "# master $M"
    $SBDIR/{{.NodeLabel}}$M/use test -e "drop table if exists t$M"
    $SBDIR/{{.NodeLabel}}$M/use test -e "create table t$M(id int not null primary key, sid int)"
    $SBDIR/{{.NodeLabel}}$M/use test -e "insert into t$M values ($M, @@server_id)"
	# $SBDIR/{{.NodeLabel}}$M/use test -e "show tables from test"
	total_tables=$((total_tables+1))
done
sleep $SLEEP_TIME
for S in $SLAVES
do
	echo "# slave $S"
	# $SBDIR/{{.NodeLabel}}$S/use test -e "show tables from test"
    found_tables=$($SBDIR/{{.NodeLabel}}$S/use test -BN -e "select count(*) from information_schema.tables where table_schema='test'")
	ok_equal $found_tables $total_tables "Slaves received tables from all masters"
done

echo "# pass: $pass"
echo "# fail: $fail"
if [ "$fail" != "0" ]
then
	exit 1
fi
exit 0
`

	ReplicationTemplates = TemplateCollection{
		"init_slaves_template": TemplateDesc{
			Description: "Initialize slaves after deployment",
			Notes:       "Can also be run after calling './clear_all'",
			Contents:    init_slaves_template,
		},
		"semi_sync_start_template": TemplateDesc{
			Description: "Starts semi synch replication ",
			Notes:       "",
			Contents:    semi_sync_start_template,
		},
		"start_all_template": TemplateDesc{
			Description: "Starts nodes in replication order (with optional mysqld arguments)",
			Notes:       "",
			Contents:    start_all_template,
		},
		"restart_all_template": TemplateDesc{
			Description: "stops all nodes and restarts them (with optional mysqld arguments)",
			Notes:       "",
			Contents:    restart_all_template,
		},
		"use_all_template": TemplateDesc{
			Description: "Execute a query for all nodes",
			Notes:       "",
			Contents:    use_all_template,
		},
		"use_all_slaves_template": TemplateDesc{
			Description: "Execute a query for all slaves",
			Notes:       "master-slave topology",
			Contents:    use_all_slaves_template,
		},
		"use_all_masters_template": TemplateDesc{
			Description: "Execute a query for all masters",
			Notes:       "master-slave topology",
			Contents:    use_all_masters_template,
		},
		"stop_all_template": TemplateDesc{
			Description: "Stops all nodes in reverse replication order",
			Notes:       "",
			Contents:    stop_all_template,
		},
		"send_kill_all_template": TemplateDesc{
			Description: "Send kill signal to all nodes",
			Notes:       "",
			Contents:    send_kill_all_template,
		},
		"clear_all_template": TemplateDesc{
			Description: "Remove data from all nodes",
			Notes:       "",
			Contents:    clear_all_template,
		},
		"status_all_template": TemplateDesc{
			Description: "Show status of all nodes",
			Notes:       "",
			Contents:    status_all_template,
		},
		"test_sb_all_template": TemplateDesc{
			Description: "Run sb test on all nodes",
			Notes:       "",
			Contents:    test_sb_all_template,
		},
		"test_replication_template": TemplateDesc{
			Description: "Tests replication flow",
			Notes:       "",
			Contents:    test_replication_template,
		},
		"check_slaves_template": TemplateDesc{
			Description: "Checks replication status in master and slaves",
			Notes:       "",
			Contents:    check_slaves_template,
		},
		"master_template": TemplateDesc{
			Description: "Runs the MySQL client for the master",
			Notes:       "",
			Contents:    master_template,
		},
		"slave_template": TemplateDesc{
			Description: "Runs the MySQL client for a slave",
			Notes:       "",
			Contents:    slave_template,
		},
		"multi_source_template": TemplateDesc{
			Description: "Initializes nodes for multi-source replication",
			Notes:       "fan-in and all-masters",
			Contents:    multi_source_template,
		},
		"multi_source_use_slaves_template": TemplateDesc{
			Description: "Runs a query for all slave nodes",
			Notes:       "group replication and multi-source topologies",
			Contents:    multi_source_use_slaves_template,
		},
		"multi_source_use_masters_template": TemplateDesc{
			Description: "Runs a query for all master nodes",
			Notes:       "group replication and multi-source topologies",
			Contents:    multi_source_use_masters_template,
		},
		"multi_source_test_template": TemplateDesc{
			Description: "Test replication flow for multi-source replication",
			Notes:       "fan-in and all-masters",
			Contents:    multi_source_test_template,
		},
		"check_multi_source_template": TemplateDesc{
			Description: "checks replication status for multi-source replication",
			Notes:       "fan-in and all-masters",
			Contents:    check_multi_source_template,
		},
	}
)
