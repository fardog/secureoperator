using System;
using System.Net;
using System.Net.NetworkInformation;
using System.Management.Automation;
using System.Collections.ObjectModel;
using System.Text;
using System.Diagnostics;
using System.Collections.Generic;
using System.IO;

namespace doh_proxy_wrapper
{
    public class NetworkingExample
    {
        static string executePath = "";
        public static void Main(string[] args)
        {
            var executeFile = System.Reflection.Assembly.GetEntryAssembly().Location;

            executePath = System.IO.Path.GetDirectoryName(executeFile);
            bool isSetup = false;
            foreach( var arg in args)
            {
                if (arg == "setup")
                {
                    isSetup = true;
                    break;
                }
            }
            if (isSetup)
            {
                setUpConfig();
                return;
            }
            var argsStr = "";
            if (args != null && args.Length > 0)
            {
                argsStr = string.Join(" ", args);
            }

            ChangeDns();

            NetworkChange.NetworkAddressChanged += new
            NetworkAddressChangedEventHandler(AddressChangedCallback);
            Console.WriteLine("Listening for address changes.");

            runCommand("doh-proxy.exe", executePath, argsStr);
        }

        static void setUpConfig()
        {
            NetworkInterface[] adapters = NetworkInterface.GetAllNetworkInterfaces();
            Dictionary<int, int> indexMap = new Dictionary<int, int>();
            int index = 1;
            for (int i = 0; i < adapters.Length; i++)
            {
                // except type: Unknown Loopback
                if (adapters[i].NetworkInterfaceType == NetworkInterfaceType.Loopback
                    || adapters[i].NetworkInterfaceType == NetworkInterfaceType.Unknown
                    || adapters[i].OperationalStatus != OperationalStatus.Up)
                {
                    continue;
                }
                Console.WriteLine($"{index++}: {adapters[i].Name}");
                indexMap.Add(index - 1, i);
            }

            List<string> NICs = new List<string>();
            while (true)
            {
                Console.WriteLine();
                Console.Write($"Input Number of NIC (0 for exit input): ");
                try
                {
                    var number = int.Parse(Console.ReadLine());
                    if (number == 0)
                    {
                        break;
                    }
                    if (NICs.Contains(adapters[indexMap[number]].Name))
                    {
                        Console.WriteLine($"Selected NICs: {string.Join(", ", NICs)}");
                        continue;
                    }
                    NICs.Add(adapters[indexMap[number]].Name);
                    Console.WriteLine($"Selected NICs: {string.Join(", ", NICs)}");
                }
                catch{
                    continue;
                }
            }
            using (StreamWriter sw = new StreamWriter(Path.Combine(executePath, "NIC.txt"), false, Encoding.GetEncoding("UTF-8")))
            {
                foreach (string s in NICs)
                {
                    sw.WriteLine(s);
                }
            }
        }

        static void runCommand(string file, string wd="", string execParams ="")
        {
            //* Create your Process
            Process process = new Process();
            if (!string.IsNullOrWhiteSpace(wd))
            {
                process.StartInfo.WorkingDirectory = wd;
            }
            process.StartInfo.FileName = file;
            if (!string.IsNullOrWhiteSpace(execParams))
            {
                process.StartInfo.Arguments = execParams;
            }
            process.StartInfo.UseShellExecute = false;
            process.StartInfo.RedirectStandardOutput = true;
            process.StartInfo.RedirectStandardError = true;
            //* Set your output and error (asynchronous) handlers
            process.OutputDataReceived += (s, e) => Console.WriteLine(e.Data);
            process.ErrorDataReceived += (s, e) => Console.WriteLine(e.Data);
            //* Start process and handlers
            process.Start();
            process.BeginOutputReadLine();
            process.BeginErrorReadLine();
            process.WaitForExit();
        }

        static void AddressChangedCallback(object sender, EventArgs e)
        {
            ChangeDns();
        }

        static void ChangeDns()
        {
            NetworkInterface[] adapters = NetworkInterface.GetAllNetworkInterfaces();
            List<string> NICs = new List<string>();
            try
            {
                using (StreamReader sr = new StreamReader(Path.Combine(executePath, "NIC.txt"), Encoding.GetEncoding("UTF-8")))
                {
                    string line = "";
                    while (!string.IsNullOrWhiteSpace((line = sr.ReadLine())))
                    {
                        NICs.Add(line);
                    }
                }
            }
            catch
            {
                Console.WriteLine($"NIC.txt file read error.");
                return;
            }
            
            foreach (NetworkInterface n in adapters)
            {

                // except type: Unknown Loopback
                if (n.NetworkInterfaceType == NetworkInterfaceType.Loopback
                    || n.NetworkInterfaceType == NetworkInterfaceType.Unknown
                    || n.OperationalStatus != OperationalStatus.Up
                    || !NICs.Contains(n.Name))
                {
                    continue;
                }
                var adapterProperties = n.GetIPProperties();
                var ips = adapterProperties.UnicastAddresses;
                var ip16 = "";
                var ip4 = "";
                foreach (var ip in ips)
                {
                    Console.WriteLine($"{n.Name} - IP: {ip.Address}");
                    if (!string.IsNullOrWhiteSpace(ip4) && !string.IsNullOrWhiteSpace(ip16))
                    {
                        break;
                    }
                    if (string.IsNullOrWhiteSpace(ip4) && 
                        ip.Address.AddressFamily == System.Net.Sockets.AddressFamily.InterNetwork)
                    {
                        ip4 = ip.Address.ToString();
                    }
                    if (string.IsNullOrWhiteSpace(ip16) &&
                        ip.Address.AddressFamily == System.Net.Sockets.AddressFamily.InterNetworkV6)
                    {
                        ip16 = ip.Address.ToString();
                    }
                }
                if (!string.IsNullOrEmpty(ip4))
                {
                    ChangeDnsInPowerShell(n.Name, ip4);
                }
                if (!string.IsNullOrEmpty(ip16))
                {
                    ChangeDnsInPowerShell(n.Name, ip16);
                }
            }
        }

        static void ChangeDnsInPowerShell(string intfaceName, string dns)
        {
            using (PowerShell PowerShellInstance = PowerShell.Create())
            {
                Console.WriteLine($"SET DNS: {intfaceName} -> {dns}");
                // use "AddScript" to add the contents of a script file to the end of the execution pipeline.
                PowerShellInstance.AddScript($"Set-DnsClientServerAddress -InterfaceAlias '{intfaceName}' -ServerAddresses ('{dns}')");

                // invoke execution on the pipeline
                PowerShellInstance.Invoke();
            }
        }
    }
}