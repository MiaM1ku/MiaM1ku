import java.io.FileDescriptor;
import java.io.FileOutputStream;
import java.io.PrintStream;
import java.lang.reflect.Constructor;
import java.lang.reflect.Method;
import java.util.List;

/**
 * Labeler: prints the user-visible label of installed apps as "package=label" lines.
 *
 * Run on device via app_process (same approach as scrcpy):
 *   CLASSPATH=/path/to/labeler.dex app_process /system/bin Labeler [pkg ...]
 *
 * Every Android API is reached through reflection so the class compiles with a plain
 * JDK (javac --release 8) and needs only the device runtime at execution time.
 * Labels follow the current system locale because they come from the system Context.
 */
public class Labeler {

    public static void main(String[] args) {
        PrintStream out;
        try {
            out = new PrintStream(new FileOutputStream(FileDescriptor.out), true, "UTF-8");
        } catch (Throwable t) {
            out = System.out;
        }
        try {
            Object pm = getPackageManager();
            Class<?> aiClass = Class.forName("android.content.pm.ApplicationInfo");
            Method getLabel = pm.getClass().getMethod("getApplicationLabel", aiClass);

            if (args.length == 0) {
                Method getInstalled = pm.getClass().getMethod("getInstalledApplications", int.class);
                List<?> apps = (List<?>) getInstalled.invoke(pm, 0);
                for (Object ai : apps) {
                    printOne(out, pm, getLabel, ai);
                }
            } else {
                Method getAppInfo = pm.getClass().getMethod("getApplicationInfo", String.class, int.class);
                for (String pkg : args) {
                    try {
                        Object ai = getAppInfo.invoke(pm, pkg, 0);
                        printOne(out, pm, getLabel, ai);
                    } catch (Throwable t) {
                        // Package missing or inaccessible: move on to the next argument.
                    }
                }
            }
            out.flush();
        } catch (Throwable t) {
            out.flush();
            System.err.println("Labeler: " + t);
            Throwable cause = t.getCause();
            if (cause != null) {
                System.err.println("Caused by: " + cause);
            }
            System.exit(1);
        }
        System.exit(0);
    }

    private static void printOne(PrintStream out, Object pm, Method getLabel, Object ai) {
        try {
            if (ai == null) {
                return;
            }
            String pkg = (String) ai.getClass().getField("packageName").get(ai);
            if (pkg == null || pkg.isEmpty()) {
                return;
            }
            Object label = getLabel.invoke(pm, ai);
            String name = normalize(label);
            if (name.isEmpty()) {
                return;
            }
            out.println(pkg + "=" + name);
        } catch (Throwable t) {
            // One broken package only drops its own line; the rest of the list still prints.
        }
    }

    private static String normalize(Object label) {
        if (label == null) {
            return "";
        }
        String s = label.toString();
        s = s.replace('\r', ' ').replace('\n', ' ').replace('\t', ' ').replace('=', ' ');
        s = s.replaceAll("\\s+", " ");
        return s.trim();
    }

    private static Object getPackageManager() throws Throwable {
        Class<?> looper = Class.forName("android.os.Looper");
        try {
            looper.getMethod("prepareMainLooper").invoke(null);
        } catch (Throwable t) {
            // A main looper may already exist; the context paths below work either way.
        }

        Class<?> atClass = Class.forName("android.app.ActivityThread");
        Object context = getSystemContext(atClass);
        return context.getClass().getMethod("getPackageManager").invoke(context);
    }

    /**
     * Obtains the system Context, trying three paths in order:
     *  1. ActivityThread.systemMain(): the primary path; it builds a full system
     *     ActivityThread whose getSystemContext() carries resources and locale.
     *  2. ActivityThread.currentActivityThread(): covers runtimes where systemMain()
     *     throws (e.g. vendor ROMs or versions that reject it outside system_server)
     *     but an ActivityThread was already attached to this process.
     *  3. new ActivityThread() + getSystemContext(): the scrcpy-style path; the
     *     private no-arg constructor yields a bare thread whose system context is
     *     enough for PackageManager queries.
     */
    private static Object getSystemContext(Class<?> atClass) throws Throwable {
        Method getSystemContext = atClass.getMethod("getSystemContext");
        Throwable firstError = null;

        // Path 1: systemMain().
        try {
            Object at = atClass.getMethod("systemMain").invoke(null);
            if (at != null) {
                Object ctx = getSystemContext.invoke(at);
                if (ctx != null) {
                    return ctx;
                }
            }
        } catch (Throwable t) {
            firstError = t;
        }

        // Path 2: an ActivityThread already present in this process.
        try {
            Object at = atClass.getMethod("currentActivityThread").invoke(null);
            if (at != null) {
                Object ctx = getSystemContext.invoke(at);
                if (ctx != null) {
                    return ctx;
                }
            }
        } catch (Throwable t) {
            if (firstError == null) {
                firstError = t;
            }
        }

        // Path 3: construct an ActivityThread directly (scrcpy approach).
        try {
            Constructor<?> ctor = atClass.getDeclaredConstructor();
            ctor.setAccessible(true);
            Object at = ctor.newInstance();
            try {
                // Register it as the current thread, as scrcpy does, so framework code
                // that calls currentActivityThread() during context setup finds it.
                java.lang.reflect.Field cur = atClass.getDeclaredField("sCurrentActivityThread");
                cur.setAccessible(true);
                cur.set(null, at);
            } catch (Throwable ignored) {
                // Field name differs on this build; getSystemContext() is still attempted.
            }
            Object ctx = getSystemContext.invoke(at);
            if (ctx != null) {
                return ctx;
            }
        } catch (Throwable t) {
            if (firstError == null) {
                firstError = t;
            }
        }

        if (firstError != null) {
            throw firstError;
        }
        throw new IllegalStateException("system context unavailable");
    }
}
