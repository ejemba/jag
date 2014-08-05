package local;

import java.util.ArrayList;
import java.util.List;

public class Foo {
    public Foo(boolean bad) throws Exception {
        if (bad == true) {
            throw new Exception();
        }
    }

    public List<String> Method1(boolean bad, List<String> x) throws Exception {
        if (bad == true) {
            throw new Exception();
        }
        ArrayList y;
        y = new ArrayList(x);
        y.add("end");
        return y;
    }

    public local.Bar Method2(local.Bar x) {
        return x;
    }
}
