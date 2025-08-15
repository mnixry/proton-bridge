using System;
using System.Collections.Generic;
using System.Linq;
using System.Net.NetworkInformation;
using System.Text;
using System.Threading.Tasks;

namespace ProtonMailBridge.UI.Tests.TestsHelper
{
    public class RetryHelper
    {
        public static void Eventually(Func<bool> condition, int retries = 10, int delaySeconds = 5)
        {
            for (int i = 0; i < retries; i++)
            {
                if (condition()) return; 

                Thread.Sleep(TimeSpan.FromSeconds(delaySeconds));
            }

            Assert.Fail();
        }

        public static void EventuallyAction(Action action, int retries = 20, int delaySeconds = 2)
        {
            Exception? lastException = null;
            for (int i = 0; i < retries; i++)
            {
                try
                {
                    action();
                    return;
                } catch (Exception e)
                {
                    lastException = e;
                    Thread.Sleep(TimeSpan.FromSeconds(delaySeconds));
                }
            }

            throw new Exception("Eventually failed after retries", lastException);
        }
    }
}
