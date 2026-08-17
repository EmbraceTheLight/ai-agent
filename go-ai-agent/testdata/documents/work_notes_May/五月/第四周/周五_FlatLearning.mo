package FlatLearning
  package Stage01Pipeline
    model Source
      parameter Real gain = 2;
      Real x(start = gain);
    equation
      x = gain + 1;
    end Source;

    model Root
      parameter Real rootGain = 3;
      Source source(gain = rootGain);
      Real y;
    equation
      y = source.x + rootGain;
    end Root;
  end Stage01Pipeline;

  package Stage02InstanceTree
    model BaseSensor
      Real value;
    equation
      value = 1;
    end BaseSensor;

    model FastSensor
      extends BaseSensor;
      parameter Real scale = 10;
    equation
      value = scale;
    end FastSensor;

    model Holder
      replaceable model Sensor = BaseSensor;
      Sensor sensor;
    end Holder;

    model Root
      Holder holder(redeclare model Sensor = FastSensor);
    end Root;
  end Stage02InstanceTree;

  package Stage03Components
    model Leaf
      parameter Real a = 1;
      Real x;
    equation
      x = a;
    end Leaf;

    model Branch
      Leaf left(a = 2);
      Leaf right(a = left.a + 1);
    end Branch;

    model Root
      Branch branch;
      Real total;
    equation
      total = branch.left.x + branch.right.x;
    end Root;
  end Stage03Components;

  package Stage04NameCompletion
    model Child
      parameter Real k = 1;
      Real x(start = k);
    equation
      x = k;
    end Child;

    model Root
      constant Real c = 4;
      parameter Real p = c + 1;
      Child child(k = p);
      Real y(start = p);
    equation
      y = child.x + c;
    end Root;
  end Stage04NameCompletion;

  package Stage05PackageProjection
    package Media
      partial package PartialMedium
        constant Integer nX = 1;
        constant Real reference_X[nX] = fill(1 / nX, nX);

        replaceable model BaseProperties
          Real X[nX](start = reference_X);
        end BaseProperties;
      end PartialMedium;

      package Water
        extends PartialMedium;
      end Water;
    end Media;

    model Reservoir
      replaceable package Medium = Media.PartialMedium;
      Medium.BaseProperties medium;
    end Reservoir;

    model Root
      replaceable package Medium = Media.Water;
      Reservoir reservoir(redeclare package Medium = Medium);
    end Root;
  end Stage05PackageProjection;

  package Stage06Evaluation
    model Root
      constant Integer n = 2;
      constant Real base = 1 + 2;
      parameter Real values[n] = fill(base, n);
      final parameter Real fixedValue = values[1] + values[2];
      Real x(start = fixedValue);
    equation
      x = fixedValue;
    end Root;
  end Stage06Evaluation;

  package Stage07MergeAndBasicType
    type Voltage = Real(quantity = "ElectricPotential", unit = "V", nominal = 1);

    model Root
      Voltage v(start = 2);
      Real plain(unit = "1", start = 2);
    equation
      v = 5;
      plain = v;
    end Root;
  end Stage07MergeAndBasicType;
end FlatLearning;
