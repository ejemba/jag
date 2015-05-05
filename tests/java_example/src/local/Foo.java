package local;

import java.net.InetAddress;
import java.util.*;

public class Foo extends SuperFoo {
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

    public String Method3(String... words) {
        String ret = new String();
        for (int i = 0; i < words.length; i++) {
            ret += words[i] + " ";
        }

        return ret;
    }

    public Set<String> Method4(Collection<String> x) {
        HashSet<String> y =  new HashSet<String>();

        Iterator<String> iter = x.iterator();
        while (iter.hasNext()) {
            y.add(iter.next());
        }
        y.add("end");

        return y;
    }

    public Map<String, String> Method5(Map<String, Integer> x) {
        HashMap<String, String> ret = new HashMap<String, String>();

        List<String> k = new ArrayList<String>(x.keySet());
        for (int i = 0; i < k.size(); i++) {
            String str;
            str = x.get(k.get(i)) + "x";
            ret.put(k.get(i), str);
        }
        return ret;
    }

    public int Method6() {
        return 42;
    }

    public int Method7() throws Exception {
        return 42;
    }

    public static int Method8() {
        return 42;
    }

    public int[] Method9() {
        int[] x = new int[2];
        x[0] = 1;
        x[1] = 2;

        return x;
    }

    public int[][] Method10() {
        int[][] x = new int[2][2];
        x[0] = Method9();
        x[1] = Method9();

        return x;
    }

    public Bar[] Method11() {
        Bar[] x = new Bar[2];
        x[0] = new Bar();
        x[1] = new Bar();

        return x;
    }

    public static int answer = 42;

    public static Bar mybar = new Bar();

    public Date Method12(Date x) {
        return x;
    }

    public List<Bar> Method13() {
        List<Bar> x = new ArrayList<Bar>();

        x.add(new Bar());
        x.add(new Bar());
        return x;
    }
}

/*
        Set<String> z = new HashSet<String>();

        z.add("hello");

        Collection<String> x = new ArrayList<String>();

        Iterator<String> w = x.iterator();
//        z.


//        y = x.iterator();

        Map<String, String> m = new HashMap<String, String>();
        m.put("a", "b");
        m.keySet()

        Map.Entry<String, String> g;
        sfsd = new HashMap<String, String>()

        InetAddress v;
        v.get
*/
