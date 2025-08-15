using System;
using System.Runtime.InteropServices;
using FlaUI.Core.AutomationElements;
using FlaUI.Core.Definitions;
using FlaUI.Core.Input;
using FlaUI.Core.Tools;
using NUnit.Framework;
using ProtonMailBridge.UI.Tests.TestsHelper;

namespace ProtonMailBridge.UI.Tests
{
    public class UIActions : TestSession
    {
        public AutomationElement AccountView => Window.FindFirstDescendant(cf => cf.ByControlType(ControlType.Pane));

        protected dynamic WaitUntilElementExistsByName(string name, TimeSpan time)
        {
            WaitForElement(() =>
            {
                RefreshWindow();
                return Window.FindFirstDescendant(cf => cf.ByName(name)) != null;
            }, time, name);

            return this;
        }
        protected AutomationElement ElementByName(string name, TimeSpan? timeout = null)
        {
            WaitUntilElementExistsByName(name, timeout ?? TestData.TenSecondsTimeout);
            return Window.FindFirstDescendant(cf => cf.ByName(name));
        }
        private void WaitForElement(Func<bool> function, TimeSpan time, string selector, string customMessage = null)
        {
            RetryResult<bool> retry = Retry.WhileFalse(
                () => {
                    try
                    {
                        App.WaitWhileBusy();
                        return function();
                    }
                    catch (COMException)
                    {
                        return false;
                    }
                },
                time, TestData.RetryInterval);

            if (!retry.Success)
            {
                if(customMessage == null)
                {
                    Assert.Fail($"Failed to get {selector} element within {time.TotalSeconds} seconds.");
                }
                else
                {
                    Assert.Fail(customMessage);
                }
            }
        }
    }
}