# frontend-flat-learning-todo
## 前端平坦化学习 TODO

模型源码位于 `myTest/flat_learning/FlatLearning.mo`。调试时把 `myTest/main.go` 中的 `flat.NewFlatten(...)` 类名替换成对应阶段入口，然后运行现有平坦化流程。

本仓库当前本地运行 `myTest/main.go` 时，记得临时设置 `GOOS=windows`。

## 阶段 01 - 主流程骨架

- [ ] 入口模型：`FlatLearning.Stage01Pipeline.Root`
- [ ] 在 `compiler/frontend/flat/flat.go:238` 断点，记录 `Flatten()` 内部阶段顺序。
- [ ] 在 `flat.go:260` 断点，确认 `expandComponents(f.Model, [])` 从根模型开始。
- [ ] 在 `flat.go:288` 断点，观察进入 `evaluate()` 前 `f.EnvElement`、`f.Env.VarEnv`、`f.Env.PackageConstantProjections` 的状态。
- [ ] 在 `flat.go:302` 断点，确认 `processEquations(f.Model)` 在统一求值之后执行；函数体入口在 `flat.go:777`。
- [ ] 在 `flat.go:318` 断点，观察 `mergeElements(f.Model, [])` 何时开始收集输出变量；函数体入口在 `flat.go:480`。
- [ ] 在 `flat.go:337` 断点，确认最终进入 `getBasicTypeSpecifier()` 做基础类型归一化；函数体入口在 `basicType.go:18`。
- [ ] 预期现象：`source.x` 和 `y` 会成为平坦化变量，方程中的引用已经带上组件路径。

## 阶段 02 - 实例树构建

- [ ] 入口模型：`FlatLearning.Stage02InstanceTree.Root`
- [ ] 在 `compiler/frontend/flat/instantiate.go:64` 断点，跟踪 `instantiateRootFlat`。
- [ ] 在 `instantiate.go:132` 断点，进入 `flatInstanceBuilder.instantiateRootOnly()`，确认根模型先被 materialize。
- [ ] 在 `instantiate_component.go:22` 断点，观察 `materializeComponentInstances()` 如何遍历非基础类型组件。
- [ ] 在 `instantiate_component.go:31` 断点，观察 `effectiveTypeSpecifier := b.materializationTypeSpecifier(...)`。
- [ ] 在 `instantiate_component.go:66` 断点，跟踪 `materializeElementInstance()`。
- [ ] 在 `instantiate_component.go:69` 断点，确认根据有效类型调用 `materializeInstanceAST(effectiveTypeSpecifier, true)`。
- [ ] 在 `instantiate_component.go:85` 和 `instantiate_component.go:86` 断点，观察 `element.Instance = prepared` 与 `element.TypeSpecifier = effectiveTypeSpecifier`。
- [ ] 预期现象：`holder.sensor` 使用重声明后的 `FastSensor` 实例，而不是默认的 `BaseSensor`。

## 阶段 03 - 组件展开

- [ ] 入口模型：`FlatLearning.Stage03Components.Root`
- [ ] 在 `compiler/frontend/flat/components.go:37` 断点，跟踪递归的 `components()` 调用。
- [ ] 在 `components.go:44` 附近观察 `newCompletionContext(...)` 的构造，重点看 parent path、resolver、env、flattener。
- [ ] 在 `components.go:82` 断点，确认普通 element 先进入 `completeElementPrefixes(element, ctx, false)`。
- [ ] 在 `components.go:147` 断点，观察普通组件通过 `expendElement(...)` 消费预构建 `Instance`。
- [ ] 在 `components.go:411` 断点，进入 `consumeElementInstanceSideEffects()`，这是递归展开子组件实例的核心入口。
- [ ] 在 `components.go:420` 和 `components.go:421` 断点，观察展开子实例前如何准备 package projection alias context。
- [ ] 预期现象：`branch.left.x`、`branch.right.x` 和 `total` 被收集为平坦化变量/方程。

## 阶段 04 - 名称补全

- [ ] 入口模型：`FlatLearning.Stage04NameCompletion.Root`
- [ ] 在 `compiler/frontend/flat/components.go:546` 断点，观察 `completeElementPrefixes()` 如何处理数组下标、binding 和 modifier。
- [ ] 在 `components.go:568` 和 `components.go:574` 断点，对比 modifier binding 与普通 element binding 的 owner scope。
- [ ] 在 `components.go:618` 断点，观察 `ElementModification` 中的 binding 如何进入名称补全。
- [ ] 在 `compiler/frontend/flat/name.go:1995` 断点，进入 `ElementBindingComponentPrefixCompletion()`。
- [ ] 在 `name.go:2038` 断点，观察当前 cref 是否先被 `replacePackageConstantProjectionReference()` 命中。
- [ ] 在 `name.go:2049` 和 `name.go:2072` 断点，观察普通 completed constant folding 的兜底路径。
- [ ] 预期现象：局部裸名会根据当前 flattened owner path 补全为合适的限定引用；可折叠常量会被替换成 literal。

## 阶段 05 - Package Projection

- [ ] 入口模型：`FlatLearning.Stage05PackageProjection.Root`
- [ ] 在 `compiler/frontend/flat/package_constant_projection.go:79` 断点，进入 `prepareElementProjectionContext()`。
- [ ] 在 `package_constant_projection.go:93` 断点，确认当前 scope 调用 `registerPackageConstantProjectionsForScope(class, parentElementList)`。
- [ ] 在 `package_constant_projection.go:23` 断点，观察当前 scope 中的 `package Medium = ...` 或 `redeclare package Medium = ...` 是否被识别成 projection alias。
- [ ] 在 `package_constant_projection.go:43` 断点，观察是否通过 `clonePackageConstantProjectionsFromVisibleAlias(...)` 从已有可见 alias 复制 projection。
- [ ] 在 `package_constant_projection.go:55` 断点，观察直接注册目标 package constants 的路径。
- [ ] 在 `package_constant_projection.go:640` 断点，条件关注 `memberName == "reference_X"`，确认 `evaluatePackageProjectionConstant(member, constEnv)` 返回数组值 `{1}`。
- [ ] 在 `package_constant_projection.go:651` 或 `package_constant_projection_alias.go:87` 断点，确认写入 `Env.PackageConstantProjections` 的 key，例如 `reservoir.Medium.reference_X`。
- [ ] 在 `package_constant_projection_alias.go:630` 断点，观察 `rememberPackageProjectionAliasContextForScope(typeSpecifier, scopePath)` 如何把 type owner 与当前组件路径关联。
- [ ] 在 `compiler/frontend/flat/name.go:260` 断点，观察 `completionContext.packageConstantProjectionReplacement(ref)` 如何根据当前 scope 可见 alias 查找 projection。
- [ ] 在 `name.go:298` 断点，确认 `packageProjectionLiteral(value)` 可以把数组值转成 `{1}`。
- [ ] 在 `name.go:312` 断点，确认 `replacePackageConstantProjectionReference()` 是 package constant projection 的消费入口。
- [ ] 在 `name.go:2038` 断点，确认 element binding 补全时先尝试 package projection literal 替换，命中后不会再走 `foldCompletedConstantReference()`。
- [ ] 当前代码的预期现象：`reservoir.Medium.reference_X` 这类 projection 在 name completion 阶段应直接替换为 literal `{1}`；不应再交给 `findConstantElementByCompletedRef()` 去查 `frontend.Library.GetClassDefinition("reservoir.Medium")`。
- [ ] 如果仍出现“未查到模型：reservoir.Medium”，回看 `name.go:900` 和 `name.go:909`，说明该引用没有被 `replacePackageConstantProjectionReference()` 提前消费。

## 阶段 06 - 求值

- [ ] 入口模型：`FlatLearning.Stage06Evaluation.Root`
- [ ] 在 `compiler/frontend/flat/flat.go:608` 断点，进入 `evaluate()`。
- [ ] 在 `flat.go:610` 附近观察 `for varName, element := range f.EnvElement` 的迭代对象。
- [ ] 单步进入 `compiler/frontend/eval/eval.go:35` 的 `EvaluateConstant()`。
- [ ] 单步进入 `compiler/frontend/eval/const.go:23` 的 `evaluateConstant()`。
- [ ] 单步进入 `compiler/frontend/eval/expression.go:11` 的 `expression()`。
- [ ] 当 `fill(base, n)` 被求值时，单步进入 `compiler/frontend/eval/func.go:627`。
- [ ] 预期现象：被追踪为 element binding 的编译期常量和 final parameter binding 会被折叠成具体值。

## 阶段 07 - 变量合并与基本类型归一化

- [ ] 入口模型：`FlatLearning.Stage07MergeAndBasicType.Root`
- [ ] 在 `compiler/frontend/flat/flat.go:318` 断点，确认 `mergeElements(f.Model, [])` 的调用时机。
- [ ] 在 `flat.go:480` 断点，进入 `mergeElements()`。
- [ ] 在 `flat.go:519` 附近观察递归处理 `element.Instance` 的路径。
- [ ] 在 `compiler/frontend/flat/basicType.go:18` 断点，进入 `getBasicTypeSpecifier()`。
- [ ] 在 `basicType.go:305` 断点，观察 `mergeArgumentList()` 如何合并别名类型 modifier。
- [ ] 在 `basicType.go:325` 附近观察 binding 归一化逻辑。
- [ ] 预期现象：最终 JSON 中类型会归一成基础类型 `Real`，但别名类型上的 `quantity`、`unit`、`nominal` 等 modifier 元数据会被合并保留。

## 完成检查清单

- [ ] 我能解释 `element.Instance` 是在哪里创建的。
- [ ] 我能解释 `element.TypeSpecifier` 什么时候被改写成有效类型。
- [ ] 我能追踪一个组件经过 `components -> expendElement -> consumeElementInstanceSideEffects -> components(child.Instance)` 的全过程。
- [ ] 我能区分普通组件名称补全、普通 completed constant folding 和 package projection literal replacement。
- [ ] 我能解释 `Env.PackageConstantProjections` 是如何注册、如何复制、如何被消费的。
- [ ] 我能解释为什么 `reservoir.Medium.reference_X` 不能交给 `frontend.Library.GetClassDefinition("reservoir.Medium")` 查源库模型。
- [ ] 我能区分 `element.Binding` 求值和 `ElementModification.Binding` 补全/求值。
- [ ] 我能解释 `VariablesList` 如何产生，以及为什么 JSON 输出前会清理 `Instance`。