within ;
package MyVisualLicensedLib
  extends Modelica.Icons.Package;

  block LicensedFilter
    "Licensed first-order filter with visible icon and diagram"
    Modelica.Blocks.Interfaces.RealInput u
      annotation (Placement(transformation(extent={{-140,-20},{-100,20}})));
    Modelica.Blocks.Interfaces.RealOutput y
      annotation (Placement(transformation(extent={{100,-10},{120,10}})));

    parameter Real k = 2 "Gain";
    parameter Modelica.Units.SI.Time T = 0.5 "Time constant";
    output Real publicState "Visible state for plotting";

  protected
    Real secretEnergy "Protected internal variable";

  equation
    T*der(publicState) = k*u - publicState;
    secretEnergy = publicState*publicState;
    y = publicState;

    annotation (
      Icon(
        coordinateSystem(preserveAspectRatio=false, extent={{-100,-100},{100,100}}),
        graphics={
          Rectangle(
            extent={{-90,70},{90,-70}},
            lineColor={28,108,200},
            fillColor={238,246,255},
            fillPattern=FillPattern.Solid,
            radius=12),
          Line(
            points={{-70,-35},{-40,-35},{-20,25},{15,25},{35,-20},{70,-20}},
            color={28,108,200},
            thickness=1.5),
          Ellipse(
            extent={{-11,11},{11,-11}},
            lineColor={217,67,67},
            fillColor={255,238,238},
            fillPattern=FillPattern.Solid,
            origin={-20,25},
            rotation=0),
          Text(
            extent={{-70,60},{70,30}},
            textColor={28,108,200},
            textString="Licensed"),
          Text(
            extent={{-60,-34},{60,-62}},
            textColor={80,80,80},
            textString="k=%k")}),
      Diagram(
        coordinateSystem(preserveAspectRatio=false, extent={{-100,-100},{100,100}}),
        graphics={
          Rectangle(
            extent={{-72,42},{72,-42}},
            lineColor={28,108,200},
            fillColor={245,250,255},
            fillPattern=FillPattern.Solid,
            radius=8),
          Text(
            extent={{-62,32},{62,10}},
            textColor={28,108,200},
            textString="Licensed dynamics"),
          Line(
            points={{-54,-20},{-24,-20},{-8,18},{18,18},{34,-12},{56,-12}},
            color={217,67,67},
            thickness=1.5),
          Text(
            extent={{-76,-58},{76,-82}},
            textColor={90,90,90},
            textString="T der(x) = k u - x")}),
      Documentation(info="<html><p>A tiny encrypted and licensed block used to test Dymola library licensing, icons, diagrams, and simulation.</p></html>"));
  end LicensedFilter;

  package Examples
    extends Modelica.Icons.ExamplesPackage;

    model DemoSimulation
      "Simulate this model after opening the encrypted library"
      Modelica.Blocks.Sources.RealExpression sine(
        y=sin(2*Modelica.Constants.pi*0.5*time))
        annotation (Placement(transformation(extent={{-78,-10},{-58,10}})));
      MyVisualLicensedLib.LicensedFilter filter(
        k=2.5,
        T=0.35)
        annotation (Placement(transformation(extent={{-20,-20},{20,20}})));
      Modelica.Blocks.Math.Gain outputScale(k=0.5)
        annotation (Placement(transformation(extent={{52,-10},{72,10}})));

    equation
      connect(sine.y, filter.u)
        annotation (Line(points={{-57,0},{-34,0},{-34,0},{-22,0}}, color={0,0,127}));
      connect(filter.y, outputScale.u)
        annotation (Line(points={{21,0},{50,0}}, color={0,0,127}));

      annotation (
        experiment(StartTime=0, StopTime=10, Tolerance=1e-6, Interval=0.01),
        Icon(
          coordinateSystem(preserveAspectRatio=false, extent={{-100,-100},{100,100}}),
          graphics={
            Rectangle(
              extent={{-86,68},{86,-68}},
              lineColor={28,108,200},
              fillColor={250,250,250},
              fillPattern=FillPattern.Solid,
              radius=10),
            Line(
              points={{-70,-28},{-40,18},{-10,-6},{20,36},{55,-18},{74,8}},
              color={217,67,67},
              thickness=1.5),
            Text(
              extent={{-70,-44},{70,-70}},
              textColor={28,108,200},
              textString="Demo")}),
        Diagram(
          coordinateSystem(preserveAspectRatio=false, extent={{-100,-100},{100,100}}),
          graphics={
            Text(
              extent={{-92,82},{92,56}},
              textColor={28,108,200},
              textString="Licensed library simulation test"),
            Text(
              extent={{-92,-58},{92,-86}},
              textColor={90,90,90},
              textString="Plot sine.y, filter.y, outputScale.y, and optionally filter.publicState")}),
        Documentation(info="<html><p>Run this example and plot <code>sine.y</code>, <code>filter.y</code>, <code>outputScale.y</code>, and <code>filter.publicState</code>.</p></html>"));
    end DemoSimulation;
  end Examples;

  annotation (
    uses(Modelica(version="4.0.0")),
    Documentation(info="<html><p>Visual licensing test package. It includes an icon, diagram annotations, and a runnable example.</p></html>"),
    Protection(
      access=Access.diagram,
      License(
        libraryKey="visual-dymola-licensing-test-key-2026-05-27",
        licenseFile="MyVisualLicensedLibAuthorization_Dymola.moe"),
      __Dymola_showDiagram=true,
      __Dymola_nestedShowDiagram=true,
      __Dymola_showDocumentation=true,
      __Dymola_nestedShowDocumentation=true,
      __Dymola_showVariables=true,
      __Dymola_showDiagnostics=true,
      __Dymola_showStatistics=true,
      __Dymola_showFlat=true));
end MyVisualLicensedLib;

