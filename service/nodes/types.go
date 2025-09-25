/**
 * @module node_types
 * @description 节点类型和分类常量定义，避免循环导入
 * @architecture 常量定义模块，提供系统级别的节点分类和类型定义
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow 无状态常量定义
 * @rules 只包含常量定义，不引用其他包，避免循环导入
 * @dependencies 无
 * @refs service/nodes/interface.go
 */

package nodes

// 节点分类常量
const (
	CategoryDataSource = "datasource" // 数据源
	CategoryTransform  = "transform"  // 数据处理
	CategoryOutput     = "output"     // 输出
	CategoryControl    = "control"    // 控制
	CategoryLogic      = "logic"      // 逻辑
	CategoryUtility    = "utility"    // 工具
)
const (
	CategoryDataSourceName = "数据源"
	CategoryTransformName  = "数据处理"
	CategoryOutputName     = "输出"
	CategoryControlName    = "控制"
	CategoryLogicName      = "逻辑"
	CategoryUtilityName    = "工具"
)
const (
	CategoryDataSourceIcon = "database"
	CategoryTransformIcon  = "transform"
	CategoryOutputIcon     = "output"
	CategoryControlIcon    = "control"
	CategoryLogicIcon      = "logic"
	CategoryUtilityIcon    = "utility"
)

// 节点类型常量
const (
	// 数据源类型
	TypePostgreSQL = "postgresql"
	TypeAPI        = "api"
	TypeFile       = "file"
	TypeStatic     = "static"
	TypeDataSource = "datasource"

	// 数据处理类型
	TypeDataFilter    = "data_filter"
	TypeDataTransform = "data_transform"
	TypeDataAggregate = "data_aggregate"
	TypeDataMap       = "data_map"
	TypeDataJoin      = "data_join"
	TypeDataGroup     = "data_group"
	TypeTransform     = "transform"

	// 输出类型
	TypeOutput = "output"
	TypeLogger = "logger"

	// 控制类型
	TypeCondition = "condition"
	TypeLoop      = "loop"
	TypeDelay     = "delay"
	TypeControl   = "control"
	TypeScript    = "script"
	TypeTimer     = "timer"
	TypeSubDAG    = "subdag"
)

// 节点类型显示名称常量
const (
	// 数据源类型显示名称
	TypePostgreSQLDisplayName = "PostgreSQL数据库"
	TypeAPIDisplayName        = "API接口"
	TypeFileDisplayName       = "文件数据源"
	TypeStaticDisplayName     = "静态数据"
	TypeDataSourceDisplayName = "通用数据源"

	// 数据处理类型显示名称
	TypeDataFilterDisplayName    = "数据过滤器"
	TypeDataTransformDisplayName = "数据转换器"
	TypeDataAggregateDisplayName = "数据聚合器"
	TypeDataMapDisplayName       = "数据映射器"
	TypeDataJoinDisplayName      = "数据连接器"
	TypeDataGroupDisplayName     = "数据分组器"
	TypeTransformDisplayName     = "通用转换器"

	// 输出类型显示名称
	TypeOutputDisplayName = "输出节点"
	TypeLoggerDisplayName = "日志输出"

	// 控制类型显示名称
	TypeConditionDisplayName = "条件节点"
	TypeLoopDisplayName      = "循环节点"
	TypeDelayDisplayName     = "延迟节点"
	TypeControlDisplayName   = "控制节点"
	TypeScriptDisplayName    = "脚本节点"
	TypeTimerDisplayName     = "定时器节点"
	TypeSubDAGDisplayName    = "子流程节点"
)

// 节点类型描述常量
const (
	// 数据源类型描述
	TypePostgreSQLDescription = "从PostgreSQL数据库中读取数据"
	TypeAPIDescription        = "通过HTTP API获取数据"
	TypeFileDescription       = "从文件系统读取数据"
	TypeStaticDescription     = "提供静态配置数据"
	TypeDataSourceDescription = "通用数据源连接器"

	// 数据处理类型描述
	TypeDataFilterDescription    = "根据条件过滤数据记录"
	TypeDataTransformDescription = "转换和处理数据字段"
	TypeDataAggregateDescription = "对数据进行聚合计算"
	TypeDataMapDescription       = "映射和转换数据结构"
	TypeDataJoinDescription      = "连接多个数据源"
	TypeDataGroupDescription     = "对数据进行分组操作"
	TypeTransformDescription     = "通用数据转换器"

	// 输出类型描述
	TypeOutputDescription = "通用数据输出节点"
	TypeLoggerDescription = "将数据写入日志"

	// 控制类型描述
	TypeConditionDescription = "基于条件控制流程分支"
	TypeLoopDescription      = "循环执行子流程"
	TypeDelayDescription     = "延迟执行指定时间"
	TypeControlDescription   = "通用流程控制节点"
	TypeScriptDescription    = "执行自定义脚本"
	TypeTimerDescription     = "定时触发执行"
	TypeSubDAGDescription    = "执行子工作流"
)

// 节点类型图标常量
const (
	// 数据源类型图标
	TypePostgreSQLIcon = "database"
	TypeAPIIcon        = "api"
	TypeFileIcon       = "file"
	TypeStaticIcon     = "data"
	TypeDataSourceIcon = "datasource"

	// 数据处理类型图标
	TypeDataFilterIcon    = "filter"
	TypeDataTransformIcon = "transform"
	TypeDataAggregateIcon = "aggregate"
	TypeDataMapIcon       = "map"
	TypeDataJoinIcon      = "join"
	TypeDataGroupIcon     = "group"
	TypeTransformIcon     = "transform"

	// 输出类型图标
	TypeOutputIcon = "output"
	TypeLoggerIcon = "log"

	// 控制类型图标
	TypeConditionIcon = "condition"
	TypeLoopIcon      = "loop"
	TypeDelayIcon     = "delay"
	TypeControlIcon   = "control"
	TypeScriptIcon    = "script"
	TypeTimerIcon     = "timer"
	TypeSubDAGIcon    = "subdag"
)

// 数据类型常量
const (
	DataTypeString  = "string"
	DataTypeNumber  = "number"
	DataTypeBoolean = "boolean"
	DataTypeArray   = "array"
	DataTypeObject  = "object"
	DataTypeAny     = "any"
)

// 组件类型常量
const (
	WidgetText     = "text"
	WidgetNumber   = "number"
	WidgetBoolean  = "boolean"
	WidgetSelect   = "select"
	WidgetTextarea = "textarea"
	WidgetPassword = "password"
	WidgetFile     = "file"
	WidgetCode     = "code"
	WidgetJSON     = "json"
)
