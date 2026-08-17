# BUG5 ---- 仿真失败，有结果文件，但是页面没有显示结果，无法绘制曲线
## 问题描述

当一个实验的所有仿真全部失败时，有些仿真可能已经产生了 mat 仿真结果，在 impact 中可以查看结果曲线，也可以下载仿真结果，但是 yslab 中，即使有仿真结果，也无法下载或查看结果。

示例模型：

```
model m1
  parameter Real failTime=5;
  Real x(
    start=1);
equation
  der(x)=-x;
algorithm
  when time > failTime then
    assert(false,"Intentional runtime failure");
  end when;
end m1;

```

设置仿真时间为 6s，单运行仿真，这样这个模型最终仿真就会失败，但是会有前 5s 的结果数据。

## 问题分析

通过检查代码，发现这个问题主要是由于后端返回仿真结果时判断有误导致的。后端获取完成仿真的实验的代码如下：

```
func SimulateResultFinishedV2View(c *gin.Context) {
	// ......
	// 获取实验下每个已完成的 case 的仿真情况数据
	for _, task := range taskInformationList {
		if task == nil {
			continue
		}
		// task 的进度为空, 或 task 的状态不为仿真完成/仿真终止/仿真失败时, 跳过该 task
		// 添加对进度的检查. 对于仿真未完成的仿真任务, 有两种情况: 1. 有部分仿真数据, 此时仿真进度 >= 30%  2. 没有仿真数据, 此时仿真进度 < 30%. 遇到情况 2 时, 跳过该仿真记录
		taskProgressRate, err := strconv.Atoi(strings.TrimSuffix(task.ProgressRate, "%"))
		if err != nil || taskProgressRate < 30 || (task.Status != config.TaskStatusFINISHED && task.Status != config.TaskStatusSTOP && task.Status != config.TaskStatusERROR) {
			continue
		}
		// ......

		// 记录单个 case 的仿真情况数据
		taskData := map[string]any{
			"index":               len(taskDataList) + 1,
			"id":                  idTaskMap[task.Id].ID,
			"create_time":         idTaskMap[task.Id].CreatedAt.Format("2006-01-02 15:04:05"),
			"simulate_status_msg": config.TaskStatusMsg[task.Status],
			"simulate_status":     statusCodeMap[task.Status],
			"simulate_start_time": "-",
			"simulate_end_time":   "-",
			"simulate_model_name": curExperiment.ModelName,
			"simulate_run_time":   "-",
			"simulate_percentage": "-",
			"experiment_name":     curExperiment.ExperimentName,
			"case_name":           idTaskMap[task.Id].CaseName,
			"animation":           curExperiment.Animation,
			"parameters":          caseParameters,
		}
		// ......
		taskDataList = append(taskDataList, taskData)
	}
	
	// 该实验有部分 case 成功完成仿真, 将仿真开始时间, 仿真结束时间, 仿真进度, 仿真运行时间, 仿真状态以及 case 完成情况写入实验数据 experimentData 中
	// 若没有一个 case 完成仿真, 则不返回该实验的数据
	if experimentStatus == config.TaskStatusFINISHED || experimentStatus == config.TaskStatusPARTIAL_FINISHED {
		// 处理并返回仿真结果
	}
}
```

这里判断整个实验的状态是否为“仿真完成”或“仿真部分完成”，遗漏了一种情况：仿真失败。仿真失败的情况下，也是可能会有仿真结果数据的。因此，遇到仿真失败的实验时，就不会返回仿真结果，尽管它存在仿真结果文件。

## 问题解决

解决这个问题比较简单，不再根据仿真状态决定是否返回仿真结果，而是根据`taskDataList`是否存在仿真数据决定：

```
func SimulateResultFinishedV2View(c *gin.Context) {
	// ......
	// 获取实验下每个已完成的 case 的仿真情况数据
	for _, task := range taskInformationList {
		if task == nil {
			continue
		}
		// task 的进度为空, 或 task 的状态不为仿真完成/仿真终止/仿真失败时, 跳过该 task
		// 添加对进度的检查. 对于仿真未完成的仿真任务, 有两种情况: 1. 有部分仿真数据, 此时仿真进度 >= 30%  2. 没有仿真数据, 此时仿真进度 < 30%. 遇到情况 2 时, 跳过该仿真记录
		taskProgressRate, err := strconv.Atoi(strings.TrimSuffix(task.ProgressRate, "%"))
		if err != nil || taskProgressRate < 30 || (task.Status != config.TaskStatusFINISHED && task.Status != config.TaskStatusSTOP && task.Status != config.TaskStatusERROR) {
			continue
		}
		// ......

		// 记录单个 case 的仿真情况数据
		taskData := map[string]any{
			"index":               len(taskDataList) + 1,
			"id":                  idTaskMap[task.Id].ID,
			"create_time":         idTaskMap[task.Id].CreatedAt.Format("2006-01-02 15:04:05"),
			"simulate_status_msg": config.TaskStatusMsg[task.Status],
			"simulate_status":     statusCodeMap[task.Status],
			"simulate_start_time": "-",
			"simulate_end_time":   "-",
			"simulate_model_name": curExperiment.ModelName,
			"simulate_run_time":   "-",
			"simulate_percentage": "-",
			"experiment_name":     curExperiment.ExperimentName,
			"case_name":           idTaskMap[task.Id].CaseName,
			"animation":           curExperiment.Animation,
			"parameters":          caseParameters,
		}
		// ......
		taskDataList = append(taskDataList, taskData)
	}
	
	// 该实验有部分 case 存在仿真结果数据, 将仿真开始时间, 仿真结束时间, 仿真进度, 仿真运行时间, 仿真状态以及 case 完成情况写入实验数据 experimentData 中
		if len(taskDataList) > 0 {
		// 处理并返回仿真结果
	}
}
```

## 引申问题

解决完这个问题后，右侧会显示仿真结果了，但是引申出了下一个问题：结果树没有显示。见 <a class="reference-link" href="BUG6%20----%20%E4%BB%BF%E7%9C%9F%E7%BB%93%E6%9E%9C%E5%AD%98%E5%9C%A8%EF%BC%8C%E4%BD%86%E6%98%AF%E7%BB%93%E6%9E%9C%E6%A0%91%E6%B2%A1%E6%9C%89%E5%B1%95%E7%A4%BA.md">BUG6 ---- 仿真结果存在，但是结果树没有展示</a>