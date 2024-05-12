using System;
using System.Threading.Tasks;
using Azure;
using Azure.Identity;
using Azure.Security.KeyVault.Secrets;

namespace Mercury
{
    interna
        l class Auth
    {
        
        const string keyVaultName = "Hecate";
        
        public static async Task<String> GetMercuryAccessKey(String secretName="MercuryAccess")
        {
            const string kvUri = "https://Hecate.vault.azure.net";

            var client = new SecretClient(new Uri(kvUri), new DefaultAzureCredential());

            try
            {
                KeyVaultSecret secret = await client.GetSecretAsync(secretName);
                return secret.Value;
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error retrieving secret: {ex.Message}");
                return null;
            }
        }
    }
}

