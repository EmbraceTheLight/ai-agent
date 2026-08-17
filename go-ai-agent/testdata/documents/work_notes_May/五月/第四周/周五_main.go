package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"
	"runtime/debug"
	"time"

	"AST/compiler/api"
	"AST/compiler/frontend/flat"

	"github.com/bytedance/sonic"
)

func main() {
	// s := time.Now()

	// go func() {
	// 	_ = http.ListenAndServe("localhost:6060", nil)
	// }()
	// time.Sleep(5 * time.Second)
	// parserFile("/Users/juzi/GolandProjects/AST/model/Buildings 9.1.0/Fluid/Chillers/Data/ElectricReformulatedEIR.mo")
	// parserFile("/Users/juzi/GolandProjects/AST/model/Modelica.mo")
	a := api.DefaultAPI{}

	_ = a.SetModelicaPath("./model")

	//
	// // result, err := c.SetModelicaPath(context.Background(), &smc.SetModelicaPathRequest{Path: "/modelicaPath/model"})
	// // result, err = c.SetModelicaPath(context.Background(), &smc.SetModelicaPathRequest{Path: "/modelicaPath"})

	// fmt.Println("设置path：", result)
	_, err := a.LoadLibrary("Modelica", "4.0.0", true)
	if err != nil {
		panic(err)
	}

	name, err := a.LoadFile("C:\\Users\\MSI\\Downloads\\ThermofluidStream(1)\\ThermofluidStream.mo", false)
	if err != nil {
		panic(err)
	}
	fmt.Println(name)
	//
	//name, err = a.LoadFile("D:\\code\\projects\\yslab\\simtekmodelicacompilerfrontend\\myTest\\flat_test_0526\\AliasTypeSpecifierMismatch.mo", false)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(name)
	//
	//name, err = a.LoadFile("D:\\code\\projects\\yslab\\simtekmodelicacompilerfrontend\\myTest\\flat_test_0528\\PackageProjectionModifierStartRepro.mo", false)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(name)

	name, err = a.LoadFile("D:\\code\\projects\\yslab\\simtekmodelicacompilerfrontend\\myTest\\flat_learning\\FlatLearning.mo", false)
	if err != nil {
		panic(err)
	}
	fmt.Println(name)
	// p, err := a.LoadFile("/Users/juzi/GolandProjects/AST/model/ParamConstMathExamples.mo", false)
	// _, err := a.LoadFile("/Users/juzi/GolandProjects/AST/model/Test.mo", false)
	// _, err = a.LoadFile("/Users/juzi/GolandProjects/AST/model/M_ForEnum.mo", false)
	// _, err = a.LoadFile("/Users/juzi/GolandProjects/AST/model/v6a.mo", false)
	// _, err = a.LoadFile("/Users/juzi/GolandProjects/AST/testsuite/flattening/element/array/elementArrayTest3/elementArray3.mo", false)
	// _, err = a.LoadFile("/Users/juzi/GolandProjects/AST/testsuite/flattening/equation/for/EquationFor1.mo", false)
	// _, err = a.LoadFile("/Users/juzi/GolandProjects/AST/testsuite/flattening/equation/for/EquationFor2.mo", false)
	// _, err = a.LoadFile("/Users/juzi/GolandProjects/AST/testsuite/flattening/equation/for/EquationFor3.mo", false)
	// _, err = a.LoadFile("/Users/juzi/GolandProjects/AST/testsuite/flattening/equation/for/EquationFor4.mo", false)
	// _, err = a.LoadFile("/Users/juzi/GolandProjects/AST/testsuite/flattening/equation/for/EquationFor5.mo", false)
	// _, err = a.LoadFile("/Users/juzi/GolandProjects/AST/testsuite/flattening/equation/for/EquationFor6.mo", false)
	// _, err = a.LoadFile("/Users/juzi/GolandProjects/AST/testsuite/flattening/equation/for/EquationFor7.mo", false)
	// _, err = a.LoadFile("model/TTT.mo", false)
	// _, err := a.LoadFile("testsuite/flattening/msl/BC1.mo", false)
	// _, err := a.LoadFile("testsuite/flattening/declarations/Annotations.mo", false)

	//
	// p, err := a.LoadFile("/Users/juzi/GolandProjects/AST/model/connect/Test.mo", false)

	// fmt.Println(p)

	// f, err := flat.NewFlatten("ParamConstMathExamples.Test1", flat.Config{})
	// f, err := flat.NewFlatten("Test.BasicRunnableExample", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Fluid.Examples.AST_BatchPlant.BatchPlant_StandardWater", flat.Config{})
	// f, err := flat.NewFlatten("MiniLibSig.Examples.Demo_SimpleChain", flat.Config{})
	// f, err := flat.NewFlatten("MiniLibSig.Examples.Demo_Optional", flat.Config{})
	// f, err := flat.NewFlatten("MiniLibSig.Examples.Demo_Redeclare", flat.Config{})
	// f, err := flat.NewFlatten("MiniLibSig.Examples.Demo_NameLookup_Import", flat.Config{})
	// f, err := flat.NewFlatten("MiniLibSig.Demo_ConditionalBlockWithEquations", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.CauerLowPassAnalog", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.ChuaCircuit", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.MultiBody.Examples.Loops.EngineV6", flat.Config{})
	// f, err := flat.NewFlatten("elementList3", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.CauerLowPassAnalog", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.ChuaCircuit", flat.Config{})
	// f, err := flat.NewFlatten("M_ForEnum", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Blocks.Examples.PID_Controller", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.First", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.MultiBody.Examples.Loops.EngineV6_analytic", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Basic.Capacitor", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Blocks.Examples.PID_Controller", flat.Config{})

	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.Filter", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.MultiBody.Examples.Systems.RobotR3.FullRobot", flat.Config{})
	// f, err := flat.NewFlatten("Test.Connect9", flat.Config{})

	// f, err := flat.NewFlatten("EquationFor1", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.MultiBody.Frames.resolve2", flat.Config{})

	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.First", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.FirstGrounded", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.Friction", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.CoupledClutches", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.LossyGearDemo1", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.LossyGearDemo2", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.LossyGearDemo3", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.ElasticBearing", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.Backlash", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.RollingWheel", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.HeatLosses", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.SimpleGearShift", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.EddyCurrentBrake", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.GenerationOfFMUs", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.OneWayClutch", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.OneWayClutchDisengaged", flat.Config{})
	//
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.SignConvention", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.InitialConditions", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.WhyArrows", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.Accelerate", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.Damper", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.Oscillator", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.Sensors", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.Friction", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.PreLoad", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.ElastoGap", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.Brake", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.HeatLosses", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.EddyCurrentBrake", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.Vehicle", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.GenerationOfFMUs", flat.Config{})

	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.CauerLowPassAnalog", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.CauerLowPassOPV", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.CauerLowPassSC", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.CharacteristicIdealDiodes", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.CharacteristicThyristors", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.ChuaCircuit", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.DifferenceAmplifier", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.HeatingMOSInverter", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.HeatingNPN_NORGate", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.HeatingPNP_NORGate", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.Resistor", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.HeatingRectifier", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.NandGate", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.OvervoltageProtection", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.Rectifier", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.ShowSaturatingInductor", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.ShowVariableResistor", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.SwitchWithArc", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.ThyristorBehaviourTest", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.AmplifierWithOpAmpDetailed", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.CompareTransformers", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.ControlledSwitchWithArc", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.SimpleTriacCircuit", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.IdealTriacCircuit", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.AD_DA_conversion", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.GenerationOfFMUs", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.ResonanceCircuits", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.InvertingAmp", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.SeriesResonance", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.ParallelResonance", flat.Config{})

	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.HeatLosses", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Thermal.FluidHeatFlow.Examples.PumpDropOut", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Thermal.FluidHeatFlow.Examples.SimpleCooling", flat.Config{})

	//f, err := flat.NewFlatten("Modelica.Thermal.FluidHeatFlow.Examples.PumpAndValve", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Thermal.FluidHeatFlow.Examples.TestOpenTank", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Thermal.FluidHeatFlow.Examples.TestCylinder", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Thermal.HeatTransfer.Examples.TwoMasses", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Thermal.HeatTransfer.Examples.Motor", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Thermal.HeatTransfer.Examples.GenerationOfFMUs", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.LossyGearDemo1", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.RollingWheel", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.GenerationOfFMUs", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.Friction", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Translational.Examples.Vehicle", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.HeatingPNP_NORGate", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.NandGate", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.Rectifier", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.ThyristorBehaviourTest", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.AD_DA_conversion", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Digital.Examples.Multiplexer", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Mechanics.Rotational.Examples.RollingWheel", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.SimpleTriacCircuit", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.AD_DA_conversion", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Batteries.Examples.BatteryDischargeCharge", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Batteries.Examples.CCCVcharging", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Batteries.Examples.ShowImpedance", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMC_DOL", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMC_YD", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMC_YDarc", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMC_Transformer", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMS_Start", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMC_Inverter", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMC_Conveyor", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMC_InverterDrive", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMC_Steinmetz", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMC_withLosses", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMC_Initialize", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMC_DCBraking", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Polyphase.Examples.TransformerYY", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Polyphase.Examples.TransformerYD", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Polyphase.Examples.Rectifier", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Polyphase.Examples.TestSensors", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Polyphase.Examples.PolyphaseRectifier", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.Rectifier1Pulse.Thyristor1Pulse_R", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierBridge2Pulse.DiodeBridge2Pulse", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierBridge2Pulse.ThyristorBridge2Pulse_DC_Drive", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierCenterTapmPulse.DiodeCenterTapmPulse", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierCenterTapmPulse.ThyristorCenterTapmPulse_RL", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierCenterTapmPulse.ThyristorCenterTapmPulse_RLV", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierCenterTapmPulse.ThyristorCenterTapmPulse_RLV_Characteristic", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierBridge2mPulse.DiodeBridge2mPulse", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierCenterTap2mPulse.DiodeCenterTap2mPulse", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierCenterTap2mPulse.ThyristorCenterTap2mPulse_R", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierCenterTap2mPulse.ThyristorCenterTap2mPulse_RL", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierCenterTap2mPulse.ThyristorCenterTap2mPulse_RLV", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierCenterTap2mPulse.ThyristorCenterTap2mPulse_RLV_Characteristic", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.DCAC.SinglePhaseTwoLevel.SinglePhaseTwoLevel_R", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.DCDC.ChopperStepDown.ChopperStepDown_R", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.DCDC.HBridge.HBridge_R", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACAC.Dimmer_R", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.SinglePhase.Examples.SeriesBode", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.SinglePhase.Examples.SeriesResonance", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.SinglePhase.Examples.ParallelResonance", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.SinglePhase.Examples.Rectifier", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.Machines.Examples.TransformerTestbench", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.Polyphase.Examples.BalancingStar", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.Polyphase.Examples.BalancingDelta", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.Polyphase.Examples.UnsymmetricalLoad", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.Polyphase.Examples.TestSensors", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Spice3.Examples.Inverter", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.Spice3.Examples.InvertersApartRecord", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.Machines.Examples.TransformerTestbench", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.Polyphase.Examples.BalancingStar", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.Polyphase.Examples.BalancingDelta", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.Polyphase.Examples.UnsymmetricalLoad", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.Polyphase.Examples.TestSensors", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Magnetic.FluxTubes.Examples.MovingCoilActuator.ForceCurrentBehaviour", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Magnetic.FundamentalWave.Examples.Components.EddyCurrentLosses", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Magnetic.FundamentalWave.Examples.Components.PolyphaseInductance", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Magnetic.QuasiStatic.FundamentalWave.Examples.Components.PolyphaseInductance", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Magnetic.QuasiStatic.FundamentalWave.Examples.Components.EddyCurrentLosses", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Magnetic.QuasiStatic.FundamentalWave.Examples.BasicMachines.InductionMachines.IMC_Characteristics", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Magnetic.QuasiStatic.FundamentalWave.Examples.BasicMachines.InductionMachines.IMC_DOL", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Magnetic.QuasiStatic.FundamentalWave.Examples.BasicMachines.InductionMachines.IMC_YD", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Magnetic.QuasiStatic.FundamentalWave.Examples.BasicMachines.InductionMachines.IMC_Transformer", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.FilterWithDifferentiation", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.Filter", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.PID_Controller", flat.Config{})
	// f, err := flat.NewFlatten("Modelica.Blocks.Examples.BooleanNetwork1", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Magnetic.FluxTubes.Examples.BasicExamples.SaturatedInductor", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Thermal.FluidHeatFlow.Examples.TwoTanks", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Digital.Examples.HalfAdder", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.Noise.Densities", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACDC.RectifierCenterTap2Pulse.DiodeCenterTap2Pulse", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Magnetic.FluxTubes.Examples.BasicExamples.QuadraticCoreAirgap", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Magnetic.FluxTubes.Examples.BasicExamples.ToroidalCoreAirgap", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Magnetic.FluxTubes.Examples.BasicExamples.ToroidalCoreQuadraticCrossSection", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.IntegerNetwork1", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Digital.Examples.NXFER", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Digital.Examples.NRXFER", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Digital.Examples.BUF3S", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Magnetic.QuasiStatic.FundamentalWave.Examples.BasicMachines.InductionMachines.IMC_withLosses", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Digital.Examples.INV3S", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Math.Distributions.Uniform.density", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.ControlledDCDrives.CurrentControlledDCPM", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.ControlledDCDrives.SpeedControlledDCPM", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.DCMachines.DCPM_withLosses", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Analog.Examples.ResonanceCircuits", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Thermal.HeatTransfer.Examples.GenerationOfFMUs", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Spice3.Examples.Spice3BenchmarkFourBitBinaryAdder", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Magnetic.FluxTubes.Examples.Hysteresis.HysteresisModelComparison", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.Noise.ImpureGenerator", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.ControlledDCDrives.Utilities.SwitchingDcDc", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.QuasiStatic.Polyphase.Examples.TestSensors", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.FilterWithRiseTime", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.Modulation", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.Noise.ActuatorWithNoise", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.Rectifier12pulseFFT", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.Rectifier6pulseFFT", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.ControlledDCDrives.PositionControlledDCPM", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.ControlledDCDrives.Utilities.DcdcInverter", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Machines.Examples.InductionMachines.IMC_Conveyor", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Polyphase.Examples.PolyphaseRectifier", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Magnetic.FundamentalWave.Examples.BasicMachines.InductionMachines.IMC_DOL", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.Noise.Utilities.Parts.MotorWithCurrentControl", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.Batteries.Examples.BatteryDischargeCharge", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Electrical.PowerConverters.Examples.ACAC.SoftStarter", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Magnetic.Fun                  damentalWave.Examples.BasicMachines.InductionMachines.ComparisonPolyphase.IMC_DOL_Polyphase", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Magnetic.FundamentalWave.Examples.BasicMachines.InductionMachines.ComparisonPolyphase.IMS_Start_Polyphase", flat.Config{})
	//f, err := flat.NewFlatten("Modelica.Blocks.Examples.Noise.Utilities.Parts.MotorWithCurrentControl", flat.Config{})

	//f, err := flat.NewFlatten("ThermofluidStream.Examples.SimpleStream", flat.Config{})
	//f, err := flat.NewFlatten("ThermofluidStream.Examples.SimpleAirCycle", flat.Config{})
	//f, err := flat.NewFlatten("ThermofluidStream.Examples.SimpleEngine", flat.Config{})
	//f, err := flat.NewFlatten("ThermofluidStream.Examples.SimpleGasTurbine", flat.Config{})
	//f, err := flat.NewFlatten("ThermofluidStream.Examples.SimpleCoolingCycle", flat.Config{})
	//f, err := flat.NewFlatten("ThermofluidStream.Examples.WaterHammer", flat.Config{})
	//f, err := flat.NewFlatten("ThermofluidStream.Examples.EspressoMachine", flat.Config{})
	//f, err := flat.NewFlatten("ThermofluidStream.Examples.VenturiPump", flat.Config{})
	//f, err := flat.NewFlatten("ThermofluidStream.Examples.VaporCycle", flat.Config{})
	//f, err := flat.NewFlatten("ThermofluidStream.Examples.HeatPump", flat.Config{})
	//f, err := flat.NewFlatten("ThermofluidStream.Examples.ReverseHeatPump", flat.Config{})

	//f, err := flat.NewFlatten("AliasTypeSpecifierMismatch.Root", flat.Config{})
	//f, err := flat.NewFlatten("PackageProjectionModifierStartRepro.Root", flat.Config{})

	f, err := flat.NewFlatten("FlatLearning.Stage01Pipeline.Root", flat.Config{})
	//f, err := flat.NewFlatten("FlatLearning.Stage02InstanceTree.Root", flat.Config{}) // 阶段 02：实例树构建
	//f, err := flat.NewFlatten("FlatLearning.Stage03Components.Root", flat.Config{}) // 阶段 03：组件展开
	//f, err := flat.NewFlatten("FlatLearning.Stage04NameCompletion.Root", flat.Config{}) // 阶段 04：名称补全
	//f, err := flat.NewFlatten("FlatLearning.Stage05PackageProjection.Root", flat.Config{}) // 阶段 05：Package Projection
	//f, err := flat.NewFlatten("FlatLearning.Stage06Evaluation.Root", flat.Config{}) // 阶段 06：求值
	//f, err := flat.NewFlatten("FlatLearning.Stage07MergeAndBasicType.Root", flat.Config{}) // 阶段 07：变量合并与基本类型归一化
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic recover:", r)
			//fmt.Println("Error model:", modelName)
			debug.PrintStack() // 打印堆栈信息
		}
	}()

	if err != nil {
		panic(err)
	}
	f.Cfg.IsBasic = true
	s := time.Now()
	err = f.Flatten()
	if err != nil {
		fmt.Println("扁平化失败：", err)
		return
	}
	s1sss := time.Since(s)

	j31, err := sonic.MarshalIndent(f, "", "  ")
	if err != nil {
		fmt.Println(err)
	}
	baseDir := "myTest/flat_test_" + getDate() + "/"
	_ = os.MkdirAll(baseDir, 0666)
	err = os.WriteFile(baseDir+f.ClassName+"-flat.json", j31, 0644)
	if err != nil {
		fmt.Println("写入文件失败:", err)
	}

	ss := time.Now()
	code, err := flat.FlattenCode(f.ClassName)
	if err != nil {
		fmt.Println(err)
	}
	sss := time.Since(ss)

	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(f.Model)

	err = os.WriteFile(baseDir+f.ClassName+"-flatCode.json", []byte(code), 0644)
	if err != nil {
		fmt.Println("写入文件失败:", err)
	}

	fmt.Println("扁平化模型名称：", f.ClassName)
	fmt.Println("扁平化时间：", s1sss)
	fmt.Println("扁平化代码获取时间：", sss)

	if time.Since(s) > time.Millisecond {
		fmt.Println("总用时time:  ", time.Since(s))
	}

	// time.Sleep(500000 * time.Second)
}

// func main() {
// 	// parserFile("/Users/juzi/GolandProjects/AST/model/Buildings 9.1.0/Fluid/Chillers/Data/ElectricReformulatedEIR.mo")
// 	// parserFile("/Users/juzi/GolandProjects/AST/model/Modelica.mo")
// 	a := api.DefaultAPI{}
//
// 	result := a.SetModelicaPath("/Users/juzi/GolandProjects/AST/model")
// 	//
// 	// // result, err := c.SetModelicaPath(context.Background(), &smc.SetModelicaPathRequest{Path: "/modelicaPath/model"})
// 	// // result, err = c.SetModelicaPath(context.Background(), &smc.SetModelicaPathRequest{Path: "/modelicaPath"})
//
// 	fmt.Println("设置path：", result)
//
// 	// p, err := a.LoadFile("/Users/juzi/GolandProjects/AST/model/ParamConstMathExamples.mo", false)
// 	// p, err := a.LoadFile("/Users/juzi/GolandProjects/AST/model/Test.mo", false)
// 	p, err := a.LoadLibrary("Modelica", "4.0.0", true)
//
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Println(p)
// 	class, _ := modelica.Library.GetClassDefinition("Modelica.Fluid.Examples.AST_BatchPlant.BaseClasses.TankWith3InletOutletArraysWithEvaporatorCondensor")
// 	s := time.Now()
// 	name.ClassInstanceTypeSpecifierCompletion(class)
// 	fmt.Println("实例化时间：", time.Since(s))
// 	fmt.Println()
// }

func getDate() string {
	curTime := time.Now()
	m, d := curTime.Month(), curTime.Day()
	return fmt.Sprintf("%02d%d", m, d)
}
