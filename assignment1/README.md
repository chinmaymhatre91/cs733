Assignment 1 - Versioned File Store

1] For serving each client a seperate handler goroutine will be called, 
   so mltiple clients can be handled at time by the server.

2] Four operations can be performed on file namely read, write, cas, and
   delete. 'read' operation reads file content and send it as reply.
   'write' writes data into file if file already exist, else creates file
   and then write data into the file. 'cas' is cpmpare-and-swap operation, 
   which compares version number for file and if they matches, writes data 
   into the file. 'delete' operation deletes the file.

3] To have a concurrency on files, with many differet handler routines 
   for different clients, trying to access/modify files I have made use 
   of channels. Routine 'handleDataAccess' handles the actual task of 
   reading, writing, modifying or deleting the files. So, the routine 
   'handleDataAccess' accepts requests for file access by listening on
   a channel. 'handleConnection' routines which are there for each client
   requests operation on files though the channel to 'handleDataAccess'
   routine. 'handleDataAccess' routine sends result to operation of a 
   seperate channel which is specific to every 'handleConnection' routine
   provided during operation request.

4] Files with expiry time should be deleted after they get expired. 
   To handle this issue I have followd a lazy apporach. So, when a file 
   read, written, or modified it is first checked for the expiry. If it is 
   expired it is deleted and operations are performed as if file were was 
   not found.

5] Upon modifying files version number for the file is updates. For simplicity
   I have incremnted the version number by one after evry modificaion to the 
   file. 

6] Input commands to the server are expected in proper specified format. If a 
   command is not in foramt, server gives command error and immidiately closes
   connection to that client.

7] I have tested the server working for diffent types of error messgaes, 
   many concurrent 'write' operation on a file, as well as many 
   concurrent 'cas' operations on a file
