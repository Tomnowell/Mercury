using Azure.Communication.Calling.WindowsClient;
using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Threading.Tasks;
using Windows.ApplicationModel;
using Windows.ApplicationModel.Core;
using Windows.Media.Core;
using Windows.Networking.PushNotifications;
using Windows.UI;
using Windows.UI.ViewManagement;
using Windows.UI.Xaml;
using Windows.UI.Xaml.Controls;
using Windows.UI.Xaml.Media;
using Windows.UI.Xaml.Navigation;


namespace Mercury
{
    public sealed partial class MainPage : Page
    {
        private string authToken = Auth.GetMercuryAccessKey().ToString();

        private CallClient callClient;
        private CallTokenRefreshOptions callTokenRefreshOptions = new CallTokenRefreshOptions(false);
        private CallAgent callAgent;
        private CommunicationCall call;

        private LocalOutgoingAudioStream micStream;

        #region Page initialization
        public MainPage()
        {
            this.InitializeComponent();
            // Additional UI customization code goes here
        }

        protected override async void OnNavigatedTo(NavigationEventArgs e)
        {
            await InitCallAgentAndDeviceManagerAsync();

            base.OnNavigatedTo(e);
        }
        #endregion

        #region UI event handlers
        private async void CallButton_Click(object sender, RoutedEventArgs e)
        {
            var callString = CalleeTextBox.Text.Trim();

            if (!string.IsNullOrEmpty(callString))
            {
                call = await StartCallAsync(callString);

                call.StateChanged += OnStateChangedAsync;
            }


        }

        private async void HangupButton_Click(object sender, RoutedEventArgs e)
        {
            var call = this.callAgent?.Calls?.FirstOrDefault();
            if (call != null)
            {
                await call.HangUpAsync(new HangUpOptions() { ForEveryone = false });
            }
        }

        private async void MuteLocal_Click(object sender, RoutedEventArgs e)
        {
            var muteCheckbox = sender as CheckBox;

            if (muteCheckbox != null)
            {
                var call = this.callAgent?.Calls?.FirstOrDefault();

                if (call != null)
                {
                    if ((bool)muteCheckbox.IsChecked)
                    {
                        await call.MuteOutgoingAudioAsync();
                    }
                    else
                    {
                        await call.UnmuteOutgoingAudioAsync();
                    }
                }

                // Update the UI to reflect the state
            }
        }
        #endregion

        #region API event handlers

        private async void OnStateChangedAsync(object sender, Azure.Communication.Calling.WindowsClient.PropertyChangedEventArgs args)
        {
            var call = sender as CommunicationCall;

            if (call != null)
            {
                var state = call.State;

                // Update the UI

                switch (state)
                {
                    case CallState.Connected:
                        {
                            await call.StartAudioAsync(micStream);

                            break;
                        }
                    case CallState.Disconnected:
                        {
                            call.StateChanged -= OnStateChangedAsync;

                            call.Dispose();

                            break;
                        }
                    default: break;
                }
            }
        }
        private async void OnIncomingCallAsync(object sender, IncomingCallReceivedEventArgs args)
        {
            var incomingCall = args.IncomingCall;

            var acceptCallOptions = new AcceptCallOptions() { };

            call = await incomingCall.AcceptAsync(acceptCallOptions);
            call.StateChanged += OnStateChangedAsync;
        }

        #endregion

        #region Helper methods

        private async Task<CommunicationCall> StartCallAsync(string acsCallee)
        {
            var options = new StartCallOptions();
            var call = await this.callAgent.StartCallAsync(new[] { new UserCallIdentifier(acsCallee) }, options);
            return call;
        }

        private async Task InitCallAgentAndDeviceManagerAsync()
        {
            this.callClient = new CallClient(new CallClientOptions()
            {
                Diagnostics = new CallDiagnosticsOptions()
                {

                    // make sure to put your project AppName
                    AppName = "Mercury",

                    AppVersion = "1.0",

                    Tags = new[] { "Calling", "ACS", "Windows" }
                }

            });

            // Set up local audio stream using the first mic enumerated
            var deviceManager = await this.callClient.GetDeviceManagerAsync();
            var mic = deviceManager?.Microphones?.FirstOrDefault();

            micStream = new LocalOutgoingAudioStream();

            var tokenCredential = new CallTokenCredential(authToken, callTokenRefreshOptions);

            var callAgentOptions = new CallAgentOptions()
            {
                DisplayName = $"{Environment.MachineName}/{Environment.UserName}",
            };

            this.callAgent = await this.callClient.CreateCallAgentAsync(tokenCredential, callAgentOptions);

            this.callAgent.IncomingCallReceived += OnIncomingCallAsync;
        }
        


        #endregion
    }
}
