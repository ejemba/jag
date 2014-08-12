package local;

import java.net.InetAddress;
import java.util.*;

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
