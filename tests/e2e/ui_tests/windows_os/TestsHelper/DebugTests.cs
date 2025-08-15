using FlaUI.Core.Capturing;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Net.NetworkInformation;
using System.Text;
using System.Threading.Tasks;

namespace ProtonMailBridge.UI.Tests.TestsHelper
{
    internal class DebugTests
    {
        public void TakeScreenshot()
        {
            string testName = TestContext.CurrentContext.Test.Name;
            string Timestamp = DateTime.Now.ToString("yyyyMMddHHmmss");
            string ScreenshotName = "Screenshot_" + testName + "_" + Timestamp + ".png";
            string Query = "%CI_PROJECT_DIR%\\tests\\e2e\\ui_tests\\windows_os\\Results\\artifacts\\Screenshots\\" + ScreenshotName;
            string ScreenshotLocation = Environment.ExpandEnvironmentVariables(Query);
            var ScreenshotFile = Capture.Screen();
            ScreenshotFile.ToFile(ScreenshotLocation);
        }
    }
}
