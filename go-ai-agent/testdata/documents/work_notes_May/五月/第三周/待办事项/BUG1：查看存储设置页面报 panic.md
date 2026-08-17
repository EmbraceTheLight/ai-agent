# BUG1：查看存储设置页面报 panic
## 问题描述

点击设置图标，接着点击存储设置，有时后端会 panic。

<figure class="image"><img style="aspect-ratio:1558/913;" src="BUG1：查看存储设置页面报 panic_image.png" width="1558" height="913"></figure>

## 问题分析

通过调试，发现后端在查询可以删除的仿真记录时出了问题。后端相关代码如下：

```
// FilterSimulationRecordByTime 返回仿真结果完成时间在给定时间之前的记录 id 列表, 以及 experimentList 中完成仿真的记录总数
func (s *Manager) FilterSimulationRecordByTime(experimentCaseList []*DataBase.YssimExperimentCase, soonerThan int) (batchCaseMap map[string][]*DataBase.YssimExperimentCase, total int) {
	// ......
	for _, taskCaseList := range batchCaseMap {
		countedTotal := false
		for _, task := range taskCaseList {
			t := taskCaseMap[task.TaskId]
			// 当前批次还有仿真任务在执行, 则将该批次从 batchCaseMap 中移除, 表示不删除该批次下的 case 的仿真结果文件, 同时不计入仿真总数
			if t.Status != config.TaskStatusFINISHED && t.Status != config.TaskStatusSTOP && t.Status != config.TaskStatusERROR {
				deleteKey := idTaskMap[t.Id].TaskBatchId
				delete(batchCaseMap, deleteKey)
				break
			}

			// ......
		}
	}

	return batchCaseMap, total
}
```

有些仿真是在一个月前完成的，有可能在`YssimExperimentCases`表中有这条记录，但是 mongoDB 中没有该仿真的记录，导致后端得到的 `t` (`taskCaseMap[task.TaskId]`) 为 nil, 在解引用时会导致 panic。

## 问题解决

增加对任务的检查机制，如果 `t` 为 nil，则认为本批次仿真是有问题的，不会执行后面的逻辑:

```
// FilterSimulationRecordByTime 返回仿真结果完成时间在给定时间之前的记录 id 列表, 以及 experimentList 中完成仿真的记录总数
func (s *Manager) FilterSimulationRecordByTime(experimentCaseList []*DataBase.YssimExperimentCase, soonerThan int) (batchCaseMap map[string][]*DataBase.YssimExperimentCase, total int) {
	batchCaseMap = make(map[string][]*DataBase.YssimExperimentCase)
	for _, experimentCase := range experimentCaseList {
		batchCaseMap[experimentCase.TaskBatchId] = append(batchCaseMap[experimentCase.TaskBatchId], experimentCase)
	}

	taskIdList, _ := GetTaskIdList(experimentCaseList)

	deleteTime := time.Now().AddDate(0, 0, -soonerThan)
	taskInfoList := GetTaskInformationList(taskIdList)
	taskCaseMap := make(map[string]*Task) // key: task_id  value: Task 任务信息结构
	for _, task := range taskInfoList {
		taskCaseMap[task.Id] = task
	}

	total = 0

	for batchId, taskCaseList := range batchCaseMap {
		// 标记该批次的结果是否可以删除, 如果该批次某些仿真遇到了不可以删除(如正在仿真, 或该仿真结束时间晚于指定的时间 soonerThan),
		// 则 keepBatch 变为 false, 可以删除的仿真批次数量不变
		keepBatch := true

		for _, task := range taskCaseList {
			// taskId 为空, 可能是该仿真任务还未下发到仿真节点, 跳过该批次仿真
			if strings.TrimSpace(task.TaskId) == "" {
				keepBatch = false
				break
			}

			t := taskCaseMap[task.TaskId]
			// 健壮性检查: 某些历史数据在 tic 服务中可能查不到, 跳过该批次仿真
			if t == nil {
				keepBatch = false
				break
			}

			// 当前批次还有仿真任务在执行, 则将 keepBatch 置为 false, 表示不删除该批次下的 case 的仿真结果文件, 同时不计入仿真总数
			if t.Status != config.TaskStatusFINISHED && t.Status != config.TaskStatusSTOP && t.Status != config.TaskStatusERROR {
				keepBatch = false
				break
			}

			// task 的仿真终止时间晚于指定的时间, 则将该批次从 batchCaseMap 中移除, 表示不删除该批次下的 case 的仿真结果文件
			if t.SimulateStopTime >= deleteTime.Unix() {
				keepBatch = false
				break
			}
		}

		// 若该批次仿真不应被删除, 则待删除批次 total 不变, 继续遍历下一个仿真批次
		if keepBatch == false {
			delete(batchCaseMap, batchId)
			continue
		}

		// 待删除批次总数 + 1
		total++
	}

	return batchCaseMap, total
}
```