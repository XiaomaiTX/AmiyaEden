# 自动权限

## 所需资料

- [esi_openai.json](./esi_openai.json)

## 基础准入(已实现)

根据新绑定角色的所属军团id，自动新增user

## ESI刷新ROLE权限

1. 查找角色的 esi_role

    - `Get character corporation roles` 合并四个role列表，去重后得到角色的esi_role

        ```json
        {
            "roles": [
                "Account_Take_1"
            ],
            "roles_at_base": [
                "Account_Take_1"
            ],
            "roles_at_hq": [
                "Account_Take_1"
            ],
            "roles_at_other": [
                "Account_Take_1"
            ]
            }
        ```

        ```text
        支持的role:
        Account_Take_1
        Account_Take_2
        Account_Take_3
        Account_Take_4
        Account_Take_5
        Account_Take_6
        Account_Take_7
        Accountant
        Auditor
        Brand_Manager
        Communications_Officer
        Config_Equipment
        Config_Starbase_Equipment
        Container_Take_1
        Container_Take_2
        Container_Take_3
        Container_Take_4
        Container_Take_5
        Container_Take_6
        Container_Take_7
        Contract_Manager
        Deliveries_Container_Take
        Deliveries_Query
        Deliveries_Take
        Diplomat
        Director
        Factory_Manager
        Fitting_Manager
        Hangar_Query_1
        Hangar_Query_2
        Hangar_Query_3
        Hangar_Query_4
        Hangar_Query_5
        Hangar_Query_6
        Hangar_Query_7
        Hangar_Take_1
        Hangar_Take_2
        Hangar_Take_3
        Hangar_Take_4
        Hangar_Take_5
        Hangar_Take_6
        Hangar_Take_7
        Junior_Accountant
        Personnel_Manager
        Project_Manager
        Rent_Factory_Facility
        Rent_Office
        Rent_Research_Facility
        Security_Officer
        Skill_Plan_Manager
        Starbase_Defense_Operator
        Starbase_Fuel_Technician
        Station_Manager
        Trader
        ```

    - 新建一个表专门处理esi_role与系统权限的关系

        > 一个系统权限可以由多个esi_role提供，一个esi_role也可以提供多个系统权限
        > Director角色始终对应admin权限
        > 允许admin在前端界面上配置esi_role与系统权限的关系

2. 根据character_corporation_title进行权限分配

    - /characters/{character_id}/titles
        > 1,2115274195,4096,'<b><color=0xFFFFFF00> 战斗狂热分子</color></b>'
        > 同样需要支持前端进行分配 没有默认数据
