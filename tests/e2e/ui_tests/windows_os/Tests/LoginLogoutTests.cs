using NUnit.Framework;
using ProtonMailBridge.UI.Tests.TestsHelper;
using ProtonMailBridge.UI.Tests.Windows;
using ProtonMailBridge.UI.Tests.Results;
using FlaUI.Core.Input;
using Microsoft.VisualStudio.TestPlatform.ObjectModel;

namespace ProtonMailBridge.UI.Tests.Tests
{
    [TestFixture]
    [Category("LoginLogoutTests")]
    public class LoginLogoutTests : TestSession
    {
        private readonly LoginWindow _loginWindow = new();
        private readonly HomeWindow _mainWindow = new();
        private readonly HomeResult _homeResult = new();
        private readonly string FreeAccountErrorText = "Bridge is exclusive to our mail paid plans. Upgrade your account to use Bridge.";
        private bool removeAccount = true;

        [Test]
        [Category("NOOP")]
        public void Noop()
        {
            TestContext.Out.WriteLine("NoOP");
            removeAccount = false;
        }

        [Test]
        [Category("DebugTests")]
        public void LoginAsFreeUser()
        {
            _loginWindow.SignIn(TestUserData.GetFreeUser());
            _homeResult.CheckIfFreeAccountErrorIsDisplayed(FreeAccountErrorText);
            removeAccount = false;
        }

        [Test]
        [Category("DebugTests")]
        public void LoginAsPaidUser()
        {
            _loginWindow.SignIn(TestUserData.GetPaidUser());
            _homeResult.CheckIfLoggedIn();
        }

        [Test]
        public void VerifyConnectedState()
        {
            _loginWindow.SignIn(TestUserData.GetPaidUser());
            _homeResult.CheckIfLoggedIn();
            _homeResult.CheckConnectedState();
        }

        [Test]
        public void VerifyAccountSynchronizingBar()
        {
            _loginWindow.SignIn(TestUserData.GetPaidUser());
            _homeResult.CheckIfSynchronizingBarIsShown();
        }

        [Test]
        public void AddAliasAddress()
        {
            _loginWindow.SignIn(TestUserData.GetAliasUser());
            _homeResult.CheckIfLoggedIn();
        }

        [Test]
        public void LoginWithMailboxPassword()
        {
            _loginWindow.SignInMailbox(TestUserData.GetMailboxUser());
            _homeResult.CheckIfLoggedIn();
            _mainWindow.SignOutAccount();
            _homeResult.CheckIfAccountIsSignedOut();
        }

        [Test]
        public void AddSameAccountTwice()
        {
            _loginWindow.SignIn(TestUserData.GetPaidUser());
            _homeResult.CheckIfLoggedIn();
            _mainWindow.AddNewAccount();
            _loginWindow.SignIn(TestUserData.GetPaidUser());
            _homeResult.CheckIfAccountAlreadySignedInIsDisplayed();
            _homeResult.ClickOkToAcknowledgeAccountAlreadySignedIn();
            _loginWindow.ClickCancelToSignIn();
            _homeResult.CheckIfLoggedIn();
        }

        [Test]
        public void AddAccountWithWrongCredentials()
        {
            _loginWindow.SignIn(TestUserData.GetIncorrectCredentialsUser());
            _homeResult.CheckIfIncorrectCredentialsErrorIsDisplayed();
            _loginWindow.ClickCancelToSignIn();
            removeAccount = false;
        }

        [Test, Order (1)]
        public void AddAccountWithEmptyCredentials()
        {
            _loginWindow.SignIn(TestUserData.GetEmptyCredentialsUser());
            _homeResult.CheckIfEnterUsernameAndEnterPasswordErrorMsgsAreDisplayed();
            _loginWindow.ClickCancelToSignIn();
            removeAccount = false;
        }

        [Test]
        public void AddSameAccountAfterBeingSignedOut()
        {
            _loginWindow.SignIn(TestUserData.GetPaidUser());
            _homeResult.CheckIfLoggedIn();
            _mainWindow.SignOutAccount();
            Wait.UntilInputIsProcessed(TimeSpan.FromSeconds(3));
            _mainWindow.ClickSignInMainWindow();
            _loginWindow.SignIn(TestUserData.GetPaidUser());
            _homeResult.CheckIfLoggedIn();
            _mainWindow.SignOutAccount();
        }

        [Test]
        public void AddDisabledAccount()
        {
            _loginWindow.SignIn(TestUserData.GetDisabledUser());
            _homeResult.CheckIfDsabledAccountErrorIsDisplayed();
            _loginWindow.ClickCancelToSignIn();
            removeAccount = false;
        }

        [Test]
        public void VerifySplitAddressesIsDisabledByDefault()
        {
            _loginWindow.SignIn(TestUserData.GetPaidUser());
            _homeResult.CheckIfSplitAddressesIsDisabledByDefault();
            Thread.Sleep(1000);
        }

        [Test]
        public void EnableAndDisableSplitAddressMode()
        {
            _loginWindow.SignIn(TestUserData.GetPaidUser());
            _mainWindow.EnableSplitAddress();
            Thread.Sleep(5000);
            _mainWindow.DisableSplitAddress();
        }

        [SetUp]
        public void TestInitialize()
        {   
            LaunchApp();
            Thread.Sleep(5000);
        }
        
        [TearDown]
        public void TestCleanup()
        {
            if (removeAccount)
            {
                _mainWindow.RemoveAccountTestCleanup();
            }
            ClientCleanup();
        }
    }
}