We've just implemented the verify command which works hapily for contract deployments, but currently doesn't work for proxies because there's no contract name! Let's try to figure out the proxy 
  in most cases by parsing the imports in the deploy script and picking the most likely candidate as the deployed contract for that proxy. This is gonna be a bit hard because we haven't done some 
  things well in the past. So let's do it this way: first lets start recording script_path in the metadata alongside the contract_path. Let's also create a debug command to fix-script-path 
  retroactively. Then let's write a utility function that, given a script file, determines the most likely contract deployed. It looks for a type(X).creationCode (take care of whitespaces) and 
  then uses the import path + remappings to figure out where the full contract path: path-to-contract.sol:contract. This function will be used in the deploy proxy command to populate contract_path
   and source_hash. Then lets implement a fix function to fix this retroactively fix-contract-path. Then we should be able to verify proxies, but in some situations maybe we couldn't determine 
  contract_path correctly. In this case let's add a contract-path argument to the verify command which will also update the deployments registry with the manual contract_path and source_hash, IF 
  the verification is successful. So that we know it's correct. Makes sense?

  Your todos where you left off:
     ☐ Create utility function to determine deployed contract from script file
     ☐ Update deploy proxy command to populate contract_path and source_hash
     ☐ Create debug fix-contract-path command
     ☐ Add contract-path argument to verify command with registry update