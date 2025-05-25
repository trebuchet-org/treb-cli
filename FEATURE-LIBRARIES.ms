Let's do something crazy complex. Let's implement linked library support and management in treb. This is gonna be a wild ride.

Overall idea:
1. Implement a LibraryDeployment contract that gets the library name in the constructor and deploys it using createx.deployCreate2 at a consistent address cross-chain. It should use vm.getCode to get the library creation code.
2. Implement a `gen library` command to generate deployment scripts for libraries. 
3. Implement a `deploy library` command to deploy a library. These don't need envs because they're global. It needs a chain tho. The deployer used will be the "default" env deployer as long as that's a private_key otherwise the user will be prompted to define a "libraries" env with a private key deployer in the toml file.
The deployment runs and the address is recorded in the deployments.json under a new "libraries" key to keep track of it so we can see what chains have libraries.
If this is a new library it's also added to the foundry.toml using the right format "src/MathLib.sol:MathLib:0x4f6127c697676BF60246962F4E751Fe3dEd5cA9f".
4. Implement a treb list --libraries command to see deployed libraries. 

After that we should be able to deploy contracts that require linked libraries automagically. 
We should also include a step to the `sync` operation for updating the foundry.toml with libraries to ensure that it's up to date.
Take extra care when editing the foundry.toml. Do it without reformatiing the whole file. Just change the library line.