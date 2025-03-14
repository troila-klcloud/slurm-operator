package consts

const (
	Slurm                  = "slurm"
	slurmPrefix            = Slurm + "-"
	SlurmUser              = "root"
	SlurmPassword          = "slurmPassword"
	SlurmctldName          = "slurmctld"
	SlurmdName             = "slurmd"
	SlurmdbdPidFile        = "/var/run/slurmdbd.pid"
	SlurmDefaultDebugLevel = "debug3"
	SlurmLogFile           = "/dev/null"

	SlurmCluster  = Slurm + "cluster"
	slurmOperator = slurmPrefix + "operator"

	SlurmctldPort = 6817
	SlurmdPort    = 6818
	SlurmdbdPort  = 6819
)

const (
	Munge = "munge"
)

const (
	ConfigMapKeySSSDConfig     = "sssd.conf"
	ConfigMapKeySlurmConfig    = "slurm.conf"
	ConfigMapKeyCGroupConfig   = "cgroup.conf"
	ConfigMapKeyGresConfig     = "gres.conf"
	ConfigMapKeySlurmdbdConfig = "slurmdbd.conf"
	ConfigMapNameSlurmConfigs  = slurmPrefix + "configs"
)

const (
	SecretMungeKeyFileName = Munge + ".key"
)

const (
	VolumeMountPathSpool       = "/var/spool"
	VolumeMountPathMungeSocket = "/run/" + Munge
)

const (
	RootPassword = "rootPassword"

	MariaDbDatabase     = "slurm_acct_db"
	MariaDbTable        = "slurm_acct_db.*"
	MariaDbUsername     = "slurm"
	MariaDbPort         = 3306
	MariaDbDefaultMyCnf = `[mariadb]
bind-address=*
default_storage_engine=InnoDB
innodb_default_row_format=DYNAMIC
innodb_buffer_pool_size=32768M
innodb_log_file_size=64M
innodb_lock_wait_timeout=900
max_allowed_packet=16M`
)

const (
	LabelNameKey   = "app.kubernetes.io/name"
	LabelNameValue = SlurmCluster

	// LabelInstanceKey value is taken from the corresponding CRD
	LabelInstanceKey = "app.kubernetes.io/instance"

	// LabelComponentKey value is taken from the corresponding CRD
	LabelComponentKey = "app.kubernetes.io/component"

	LabelPartOfKey   = "app.kubernetes.io/part-of"
	LabelPartOfValue = slurmOperator

	LabelManagedByKey   = "app.kubernetes.io/managed-by"
	LabelManagedByValue = slurmOperator
)

const (
	DatabaseConditionType         = "DatabaseReady"
	AccountingNodeConditionType   = "AccountingNodeAvailable"
	LoginNodeConditionType        = "LoginNodeAvailable"
	ControllerNodeConditionType   = "ControllerNodeAvailable"
	ComputingNodeSetConditionType = "ComputingNodeSet%sAvailable"
	WebNodeConditionType          = "WebNodeAvailable"
)

const (
	ConditionStatusTrue    = "True"
	ConditionTypeAvailable = "Available"
	ConditionTypeReady     = "Ready"
)

const (
	KindPersistentVolumeClaim = "PersistentVolumeClaim"
	KindDeployment            = "Deployment"
	KindStatefulset           = "Statefulset"
	KindService               = "Service"
	KindMariaDB               = "MariaDB"
	KindGrant                 = "Grant"
	KindPod                   = "Pod"
)
