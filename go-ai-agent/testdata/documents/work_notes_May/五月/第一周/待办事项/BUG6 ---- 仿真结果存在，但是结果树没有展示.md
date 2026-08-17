# BUG6 ---- 仿真结果存在，但是结果树没有展示
## 问题描述

示例模型同 <a class="reference-link" href="BUG5%20----%20%E4%BB%BF%E7%9C%9F%E5%A4%B1%E8%B4%A5%EF%BC%8C%E6%9C%89%E7%BB%93%E6%9E%9C%E6%96%87%E4%BB%B6%EF%BC%8C%E4%BD%86%E6%98%AF%E9%A1%B5%E9%9D%A2%E6%B2%A1%E6%9C%89%E6%98%BE%E7%A4%BA%E7%BB%93%E6%9E%9C%EF%BC%8C%E6%97%A0%E6%B3%95%E7%BB%98%E5%88%B6%E6%9B%B2%E7%BA%BF.md">BUG5 ---- 仿真失败，有结果文件，但是页面没有显示结果，无法绘制曲线</a>，仿真失败的模型有结果文件，但是没有结果树

## 问题分析

这个问题是由于解析结果树时，默认读取的是`result_init.xml`文件，但是这个是仿真成功结束之后才会有的，仿真未成功时，该文件其实也存在，只是其名称为`modelDescription.xml`，当仿真成功结束之后，才会将其重命名为`result_init.xml`，因此，需要加一个判断逻辑。

## 问题解决

这是一个临时方案：

```
func SimulateResultTreeV2View(c *gin.Context) {
	// ......
		// 判断当前仿真任务进度是否 ≥ 30%, 若是, 则表示有结果文件, 解析当前结果文件, 获得结果树数据
		// 否则跳过当前仿真任务
		if taskProgressRate >= 30 {
			resultFilePath := taskIdMap[task.Id].SimulateModelResultPath
			if experimentList[0].SimulateType == "DM" {
				// DM生成的fmu解压后的xml文件
				// FIXME: 当前仿真只有在仿真成功完成后, 会将 modelDescription.xml 重命名为 result_init.xml
				// (临时解决办法) 这里做一个兜底判断, 当仿真未成功完成, 但是有结果文件, 这种情况下解析 modelDescription.xml 文件
				if fileOperation.Exists(resultFilePath+"result_init.xml") == true {
					result = simulate.DymolaSimulationResultTree(resultFilePath+"result_init.xml", parentNode, keyWords, experimentList[0].SimulateType, "all", exportVariableList)
				} else {
					result = simulate.DymolaSimulationResultTree(resultFilePath+"modelDescription.xml", parentNode, keyWords, experimentList[0].SimulateType, "all", exportVariableList)
				}
			} else {
				// OMC仿真完输出的xml文件
				result = simulate.SimulationResultTree(resultFilePath, parentNode, keyWords, experimentList[0].SimulateType, exportVariableList)
			}
			// ......
		}
	// ......		
}
```