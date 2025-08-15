using FlaUI.Core.AutomationElements;
using FlaUI.Core.Definitions;
using ProtonMailBridge.UI.Tests.TestsHelper;
using FlaUI.Core.Input;
using System.DirectoryServices;

namespace ProtonMailBridge.UI.Tests.Results
{
    public class HomeResult : UIActions
    {
        private Button SignOutButton => AccountView.FindFirstDescendant(cf => cf.ByControlType(ControlType.Button).And(cf.ByName("Sign out"))).AsButton();
        private AutomationElement NotificationWindow => Window.FindFirstDescendant(cf => cf.ByControlType(ControlType.Window));
        private TextBox FreeAccountErrorText => NotificationWindow.FindFirstDescendant(cf => cf.ByControlType(ControlType.Text)).AsTextBox();
        private TextBox SignedOutAccount => AccountView.FindFirstDescendant(cf => cf.ByControlType(ControlType.Text)).AsTextBox();
        private TextBox AlreadySignedInText => NotificationWindow.FindFirstDescendant(cf => cf.ByControlType(ControlType.Text)).AsTextBox();
        private Button OkToAcknowledgeAccountAlreadySignedIn => NotificationWindow.FindFirstDescendant(cf => cf.ByControlType(ControlType.Button).And(cf.ByName("OK"))).AsButton();
        private AutomationElement[] TextFields => Window.FindAllDescendants(cf => cf.ByControlType(ControlType.Text));
        private TextBox SynchronizingField => TextFields[4].AsTextBox();
        private TextBox AccountDisabledErrorText => Window.FindAllDescendants(cf => cf.ByControlType(ControlType.Text)).FirstOrDefault(e =>!string.IsNullOrEmpty(e.Name) && e.Name.IndexOf("This account has been suspended due to a potential policy violation.", StringComparison.OrdinalIgnoreCase) >= 0)?.AsTextBox();
        private TextBox IncorrectLoginCredentialsErrorText => Window.FindFirstDescendant(cf => cf.ByControlType(ControlType.Text).And(cf.ByName("Incorrect login credentials"))).AsTextBox();
        private TextBox EnterEmailOrUsernameErrorText => Window.FindFirstDescendant(cf => cf.ByControlType(ControlType.Text).And(cf.ByName("Enter email or username"))).AsTextBox();
        private TextBox EnterPasswordErrorText => Window.FindFirstDescendant(cf => cf.ByControlType(ControlType.Text).And(cf.ByName("Enter password"))).AsTextBox();
        private TextBox ConnectedStateText => Window.FindFirstDescendant(cf => cf.ByControlType(ControlType.Text).And(cf.ByName("Connected"))).AsTextBox();
        private CheckBox SplitAddressesToggle => AccountView.FindFirstDescendant(cf => cf.ByControlType(ControlType.CheckBox).And(cf.ByName("Split addresses toggle"))).AsCheckBox();

        

        public HomeResult CheckConnectedState()
        {
            RetryHelper.Eventually(() => ConnectedStateText.IsAvailable);
            return this;
        }
        public HomeResult CheckIfLoggedIn()
        {
            RetryHelper.Eventually(() => SignOutButton.IsAvailable);
            return this;
        }
        public HomeResult CheckIfSynchronizingBarIsShown()
        {
            RetryHelper.Eventually(() => SynchronizingField.IsAvailable && SynchronizingField.Name.StartsWith("Synchronizing"));
            return this;
        }
        public HomeResult CheckIfFreeAccountErrorIsDisplayed(string ErrorText)
        {
            RetryHelper.Eventually(() => FreeAccountErrorText.Name == ErrorText);
            return this;
        }
        public HomeResult CheckIfAccountIsSignedOut()
        {
            RetryHelper.Eventually(() => SignedOutAccount.IsAvailable);
            return this;
        }
        public HomeResult CheckIfAccountAlreadySignedInIsDisplayed()
        {

            RetryHelper.Eventually(() => AlreadySignedInText.IsAvailable);
            return this;
        }
        public HomeResult ClickOkToAcknowledgeAccountAlreadySignedIn ()
        {
            OkToAcknowledgeAccountAlreadySignedIn.Click();
            return this;
        }

        public HomeResult CheckIfIncorrectCredentialsErrorIsDisplayed()
        {
            Assert.That(IncorrectLoginCredentialsErrorText.IsAvailable, Is.True);
            return this;
        }

        public HomeResult CheckIfEnterUsernameAndEnterPasswordErrorMsgsAreDisplayed()
        {
            RetryHelper.Eventually(() => EnterEmailOrUsernameErrorText.IsAvailable && EnterPasswordErrorText.IsAvailable);
            return this;
        }

        public HomeResult CheckIfDsabledAccountErrorIsDisplayed()
        {
            RetryHelper.Eventually(() => AccountDisabledErrorText.IsAvailable);
            return this;
        }

        public HomeResult CheckIfNotificationTextIsShown()
        {
            RetryHelper.Eventually(() => AlreadySignedInText.IsAvailable);
            return this;
        }

        public HomeResult CheckIfSplitAddressesIsDisabledByDefault()
        {
            RetryHelper.Eventually(() =>
            {
                bool isNotToggled = SplitAddressesToggle.IsToggled == null || !(bool)SplitAddressesToggle.IsToggled;
                return isNotToggled;
            });
            return this;
        }
    }
}