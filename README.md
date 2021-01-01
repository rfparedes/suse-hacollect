# suse-hacollect
This script is designed to collect data from an SAP/HA cluster running SLES. It collects all data needed by support teams to analyze most cluster issues. What does it install and collect?

    Installs supportutils-plugin-ha-sap (if available) on all nodes
    Collects supportconfig from all nodes in cluster
    Collects hb_report

It is meant to solve the difficulty obtaining the correct data from customers, which is necessary when diagnosing complex cluster issues. It also has a supportconfig hang detector (which happens sometimes) so that user intervention is not needed to fix.

You only need to run this on one of the nodes. It will collect the data locally on that node, compress it, and even upload it directly to SUSE via https if you include the -u option. If you choose not to automatically upload, you will be presented the final location of the tarball once complete which you can manually transfer.

The script accepts several options but only requires the start date in the format yyyy-mm-dd based on when the issue occurred. If the issue occurred on 2020-08-15 at 12:35, then for example, from the primary node in the cluster, run:
suse-hacollect -f 2020-08-15

Since there may be things going wrong with the cluster before they are ultimately noticed, leave enough time in the start date to account for this. For example, if a node was fenced on 2020-09-12 at 00:23, then use the previous day as the start date and run:
suse-hacollect -f 2020-09-11