using System;
using System.Threading;
using System.Management.Automation;
using System.Management.Automation.Runspaces;
using FlaUI.Core.AutomationElements;
using FlaUI.Core;
using FlaUI.UIA3;
using ProtonMailBridge.UI.Tests.TestsHelper;
using FlaUI.Core.Input;
using System.Diagnostics;
using System.Drawing.Text;
using System.Collections.ObjectModel;
using FlaUI.Core.Tools;

namespace ProtonMailBridge.UI.Tests
{
    public class TestSession
    {

        public static Application App;
        protected static Application Service;
        protected static Window Window;
        protected static Window ChromeWindow;
        protected static Window FileExplorerWindow;
        protected static Runspace? myRunSpace;
        protected static string? bridgeDownloadURL;
        protected static string downloadEnvVariable = "BRIDGE_DOWNLOAD_URL";
        private static readonly DebugTests debugTests = new();
        protected static string atlasEnvironment = "https://mail-api.proton.pink/";


        protected static void ClientCleanup()
        {
            var outcome = TestContext.CurrentContext.Result.Outcome.Status;
            if (outcome == NUnit.Framework.Interfaces.TestStatus.Failed)
            {
                try
                {
                    debugTests.TakeScreenshot();
                }
                catch (Exception ex)
                {
                    TestContext.Out.WriteLine(ex.ToString());
                }
            }

            try
            {
                App.Kill();
            }
            catch (Exception ex)
            {
                TestContext.Out.WriteLine(ex.ToString());
            }

            try
            {
                App.Dispose();
            }
            catch (Exception ex)
            {
                TestContext.Out.WriteLine(ex.ToString());
            }

            // Give some time to properly exit the app
            Thread.Sleep(10000);
            try
            {
                RemoveBridgeCredentials();
            }
            catch (Exception ex)
            {
                TestContext.Out.WriteLine($"Failed to remove Bridge credentials: {ex}");
            }

        }

        public static void switchToFileExplorerWindow()
        {
            var _automation = new UIA3Automation();
            var desktop = _automation.GetDesktop();

            var _explorerWindow = desktop.FindFirstDescendant(cf => cf.ByClassName("CabinetWClass"));

            // If the File Explorer window is not found, fail the test
            if (_explorerWindow == null)
            {
                throw new Exception("File Explorer window not found.");
            }

            // Cast the found element to a Window object
            FileExplorerWindow = _explorerWindow.AsWindow();

            // Focus on the File Explorer window
            FileExplorerWindow.Focus();
        }

        public static void switchToChromeWindow()
        {
            var _automation = new UIA3Automation();
            var desktop = _automation.GetDesktop();

            var _chromeWindow = desktop.FindFirstDescendant(cf => cf.ByClassName("Chrome_WidgetWin_1"));

            // If the Chrome window is not found, fail the test
            if (_chromeWindow == null)
            {
                throw new Exception("Google Chrome window not found.");
            }

            // Cast the found element to a Window object
            ChromeWindow = _chromeWindow.AsWindow();

            // Focus on the Chrome window
            ChromeWindow.Focus();
        }

        protected static void RefreshWindow(TimeSpan? timeout = null)
        {
            Window = null;
            TimeSpan refreshTimeout = timeout ?? TestData.ThirtySecondsTimeout;
            RetryResult<Window> retry = Retry.WhileNull(
                () =>
                {
                    try
                    {
                        Window = App.GetMainWindow(new UIA3Automation(), refreshTimeout);
                    }
                    catch (System.TimeoutException)
                    {
                        // Ignore
                    }
                    return Window;
                },
                refreshTimeout, TestData.RetryInterval);

            if (!retry.Success)
            {
                Assert.Fail($"Failed to refresh window in {refreshTimeout.TotalSeconds} seconds.");
            }
        }


        public static void LaunchApp()
        {
            TestContext.Out.WriteLine($"[RUNNING TEST] {TestContext.CurrentContext.Test.FullName}");
            System.Environment.SetEnvironmentVariable("BRIDGE_HOST_URL", $"{atlasEnvironment}");
            string appExecutable = TestData.AppExecutable;
            Application.Launch(appExecutable);
            Wait.UntilInputIsProcessed(TestData.FiveSecondsTimeout);
            Retry.WhileException( () =>
            {
                App = Application.Attach("bridge-gui.exe");
            }, TimeSpan.FromSeconds(60), null, true);
            RefreshWindow(TestData.OneMinuteTimeout);
            Window.Focus();
        }

        private static RetryResult<bool> WaitUntilAppIsRunning()
        {
            RetryResult<bool> retry = Retry.WhileFalse(
                () =>
                {
                    Process[] pname = Process.GetProcessesByName("Proton Mail Bridge");
                    return pname.Length > 0;
                },
                TimeSpan.FromSeconds(30), TestData.RetryInterval);

            return retry;
        }
        public static void CreateRunSpace()
        {
            bridgeDownloadURL = Environment.GetEnvironmentVariable($"{downloadEnvVariable}");
            myRunSpace = RunspaceFactory.CreateRunspace();
            myRunSpace.Open();
            Pipeline cmd = myRunSpace.CreatePipeline($"New-Item env:{downloadEnvVariable} -Value {bridgeDownloadURL} -Force");
            cmd.Invoke();
            cmd = myRunSpace.CreatePipeline(@"Set-Location $env:CI_PROJECT_DIR\tests\e2e\ui_tests\windows_os\InstallerScripts");
            cmd.Invoke();
            cmd = myRunSpace.CreatePipeline(@"Set-ExecutionPolicy -ExecutionPolicy Unrestricted -Scope CurrentUser");
            cmd.Invoke();
        }

        public static Collection<PSObject>? InstallBridge()
        {
            CreateRunSpace();
            if (myRunSpace is not null)
            {
                Pipeline cmd = myRunSpace.CreatePipeline("Get-Location");
                cmd.Invoke();
                cmd = myRunSpace.CreatePipeline(@".\Get-BridgeInstaller.ps1");
                var objects = cmd.Invoke();

                return objects;
            }

            return null;
        }

        public static Collection<PSObject>? UninstallBridge()
        {
            CreateRunSpace();
            if (myRunSpace is not null)
            {
                Pipeline cmd = myRunSpace.CreatePipeline(@".\Remove-Bridge.ps1");
                var objects = cmd.Invoke();

                return objects;
            }

            return null;
        }

        public static Collection<PSObject>? RemoveBridgeCredentials()
        {
            CreateRunSpace();
            if (myRunSpace is not null)
            {
                Pipeline cmd = myRunSpace.CreatePipeline(@".\Remove-BridgeCredentials.ps1");
                var objects = cmd.Invoke();

                return objects;
            }

            return null;
        }
    }
}